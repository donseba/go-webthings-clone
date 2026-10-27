package main

import (
	"bytes"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	webthings "github.com/donseba/go-webthings"
	"github.com/donseba/go-webthings/frontend"
	"github.com/donseba/go-webthings/sitetest"
)

// TestTemplates starts the website with its own templates, translations and
// modules, adds a page for every module and some content, and renders every
// kind of page in each theme. A mistake in a template, such as a field that
// does not exist, fails here instead of on the live site.
func TestTemplates(t *testing.T) {
	for _, theme := range []string{"default", "editorial"} {
		t.Run(theme, func(t *testing.T) {
			s := startSite(t, theme)
			visitor := s.Visitor()

			for path, parts := range map[string][]string{
				"/nl/":                       {`<main id="content"`},
				"/nl/nieuws":                 {"Havenfestival"},
				"/nl/nieuws/havenfestival":   {"<h1>Havenfestival</h1>", "Boten en muziek."},
				"/nl/blog":                   {"Havenfestival"},
				"/nl/blog/havenfestival":     {"<h1>Havenfestival</h1>"},
				"/nl/agenda":                 {"Proeverij"},
				"/nl/agenda/proeverij":       {"<h1>Proeverij</h1>", "14:30"},
				"/nl/aanmelden":              {`<main id="content"`},
				"/nl/gastenboek":             {`<main id="content"`, `maxlength="100"`, `maxlength="5000"`, "Bericht"},
				"/nl/peilingen":              {`<main id="content"`},
				"/nl/nieuwsbrief":            {`<main id="content"`},
				"/nl/search?q=havenfestival": {"Havenfestival"},
				"/en/":                       {`<main id="content"`},
				"/sitemap.xml":               {"/nl/nieuws/havenfestival"},
				"/theme/site.css":            {},
			} {
				visitor.Expect(path, http.StatusOK, parts...)
			}

			// The accent colour: the default theme's colour is a custom
			// property of the theme's values stylesheet ({{themeStyle}}),
			// the editorial theme's choice of four an attribute of <html>
			// that site.css reads. No page has a style attribute or a
			// <style> element, which a Content-Security-Policy without
			// 'unsafe-inline' ignores.
			if theme == "default" {
				_, home := visitor.Get("/nl/")

				link := regexp.MustCompile(`<link rel="stylesheet" href="(/theme/default/values\.css\?v=[0-9a-f]+)" integrity="sha384-`).
					FindStringSubmatch(home)
				if link == nil {
					t.Fatalf("no values stylesheet:\n%s", home)
				}

				visitor.Expect(link[1], http.StatusOK, "--accent: #2563eb;")
			} else {
				_, home := visitor.Get("/nl/")

				// The width of the text column is a number setting, written
				// with its unit.
				link := regexp.MustCompile(`<link rel="stylesheet" href="(/theme/editorial/values\.css\?v=[0-9a-f]+)" integrity="sha384-`).
					FindStringSubmatch(home)
				if link == nil || !strings.Contains(home, `data-accent="vermilion"`) {
					t.Fatalf("no accent or values stylesheet:\n%s", home)
				}

				visitor.Expect(link[1], http.StatusOK, "--measure: 42rem;")
			}

			inline := regexp.MustCompile(`(?i)\sstyle=|<style[\s>]`)

			for _, path := range []string{
				"/nl/", "/nl/nieuws", "/nl/nieuws/havenfestival", "/nl/blog", "/nl/blog/havenfestival", "/nl/agenda",
				"/nl/agenda/proeverij", "/nl/aanmelden", "/nl/gastenboek", "/nl/peilingen", "/nl/nieuwsbrief",
				"/nl/search?q=havenfestival", "/nl/bestaat-niet",
			} {
				if _, body := visitor.Get(path); inline.MatchString(body) {
					t.Errorf("%s has an inline style: %q", path, inline.FindAllString(body, -1))
				}
			}

			visitor.Expect("/nl/bestaat-niet", http.StatusNotFound, `<main id="content"`)

			// The guestbook form posts as before; its fields come from the module.
			message := url.Values{
				"subject": {"comments"},
				"name":    {"Joris"},
				"body":    {"Mooie haven!"},
				"back":    {"/nl/gastenboek"},
			}

			if _, _, landed := visitor.Send("/_/comments/post", message); landed != "/nl/gastenboek?comments=sent" &&
				landed != "/nl/gastenboek?comments=pending" {
				t.Fatalf("message: %s", landed)
			}

			message.Set("name", "")

			if _, _, landed := visitor.Send("/_/comments/post", message); landed != "/nl/gastenboek?comments=invalid" {
				t.Fatalf("message without a name: %s", landed)
			}
		})
	}
}

// TestEditorialTheme checks what the editorial theme adds: the tags of the
// head region, its stylesheets and scripts with their integrity, the
// regions and fragments htmx swaps, and pages without style attributes, so
// they work under a Content-Security-Policy without 'unsafe-inline'.
func TestEditorialTheme(t *testing.T) {
	s := startSite(t, "editorial")
	visitor := s.Visitor()

	for path, parts := range map[string][]string{
		"/nl/": {
			`data-accent="vermilion"`, `data-text="serif"`, `class="skip-link" href="#content"`,
			`hreflang="x-default"`, `<script type="application/ld+json">`,
			`/theme/editorial/site.css?v=`, `/theme/editorial/site.js?v=`, `integrity="sha384-`,
			`<script type="importmap"`, `"htmx": "/assets/vendor/htmx/htmx.esm.min.js?v=`,
			`"partial-interactions": "/assets/partial-interactions.js?v=`, `data-partial-reload-on-error=""`, `hx-status:5xx:inherited="swap:none push:false"`,
			`id="menu"`, `id="menu-`, `id="languages"`, `id="editor"`, `id="newsletter"`, "Laatste nieuws", "Binnenkort",
		},
		"/nl/nieuws": {`type="application/atom+xml"`, `type="application/rss+xml"`, `/nl/nieuws/feed.atom`},
		"/nl/nieuws/havenfestival": {
			`"@type":"Article"`, `id="comments"`, `name="subject"`, "Laat een reactie achter",
			`aria-hidden="true" class="form-honeypot">`,
		},
		"/nl/agenda/proeverij":       {`"@type":"Event"`, "Europe/Amsterdam"},
		"/nl/search?q=havenfestival": {`id="results"`, "2 resultaten voor"},
		"/nl/nieuwsbrief":            {`id="newsletter"`, `name="email"`},
	} {
		visitor.Expect(path, http.StatusOK, parts...)

		if _, body := visitor.Get(path); strings.Contains(body, ` style="`) {
			t.Errorf("%s has a style attribute", path)
		}
	}

	visitor.Expect("/nl/bestaat-niet", http.StatusNotFound, "Deze pagina bestaat niet")
	visitor.Expect("/theme/editorial/site.css", http.StatusOK, `[data-accent="forest"]`)

	// A category's list, also one without posts, has the category as its
	// heading and the list's own title above it.
	category := s.Submit("/admin/news-categories/new?locale=nl", url.Values{"Title": {"Haven"}, "Slug": {"haven"}})
	s.Publish(category + "?locale=nl")
	visitor.Expect("/nl/nieuws/category/haven", http.StatusOK,
		`<p class="kicker"><a href="/nl/nieuws">Nieuws</a></p>`, "<h1>Haven</h1>", `/nl/nieuws/category/haven/feed.atom`)

	// The import map's files: htmx, vendored in assets/vendor, and
	// go-partial's interactions client, which core adds; site.js imports
	// both.
	visitor.Expect("/theme/editorial/site.js", http.StatusOK, `import "htmx";`, `import "partial-interactions";`)
	visitor.Expect("/assets/vendor/htmx/htmx.esm.min.js", http.StatusOK)
	visitor.Expect("/assets/partial-interactions.js", http.StatusOK, "reloadOnError")

	// Live search asks for the results, which come with the regions out of
	// band, so the tab's title and the language links follow the query.
	if status, body := visitor.Fragment("/nl/search?q=havenfestival", "results"); status != http.StatusOK ||
		!strings.HasPrefix(strings.TrimSpace(body), `<div class="results" id="results"`) || strings.Contains(body, "<html") ||
		strings.Contains(body, `<main`) || !strings.Contains(body, "<title>havenfestival · ") ||
		!strings.Contains(body, `/en/search?q=havenfestival`) {
		t.Fatalf("results fragment: %d\n%s", status, body)
	}

	// A comment posted with htmx returns to the post, which answers with the
	// comments alone; a post with mistakes keeps what the visitor wrote.
	_, post := visitor.Get("/nl/nieuws/havenfestival")

	subject := subjectPattern.FindStringSubmatch(post)
	if subject == nil {
		t.Fatalf("no comment subject on the post\n%s", post)
	}

	comment := url.Values{
		"subject": {subject[1]},
		"name":    {""},
		"body":    {"Prachtig verhaal."},
		"back":    {"/nl/nieuws/havenfestival"},
	}

	_, body, _ := visitor.SendHTMX("/_/comments/post", comment, "comments")
	if !strings.HasPrefix(strings.TrimSpace(body), `<section class="comments" id="comments"`) ||
		!strings.Contains(body, "Prachtig verhaal.") || strings.Contains(body, "<html") {
		t.Fatalf("comment with a mistake: %s", body)
	}

	comment.Set("name", "Joris")

	_, body, _ = visitor.SendHTMX("/_/comments/post", comment, "comments")
	if !strings.Contains(body, "Bedankt") {
		t.Fatalf("comment: %s", body)
	}

	// So does the newsletter signup in the footer, from any page.
	signup := url.Values{"locale": {"nl"}, "email": {"lezer@example.test"}, "back": {"/nl/agenda"}}

	_, body, _ = visitor.SendHTMX("/_/newsletter/subscribe", signup, "newsletter")
	if !strings.HasPrefix(strings.TrimSpace(body), `<section class="signup signup--footer" id="newsletter"`) ||
		!strings.Contains(body, "kijk in je inbox") {
		t.Fatalf("newsletter: %s", body)
	}
}

// subjectPattern finds the subject of a comment form: the post's ID.
var subjectPattern = regexp.MustCompile(`name="subject"[^>]*value="([^"]+)"`)

// startSite starts the website with its own templates, translations and
// modules and theme as the active theme, places every module on a page and
// publishes a post in the news and the blog and an event.
func startSite(t *testing.T, theme string) *sitetest.Site {
	t.Helper()

	translations, err := frontend.Translations("translations")
	if err != nil {
		t.Fatal(err)
	}

	imports, err := importMap()
	if err != nil {
		t.Fatal(err)
	}

	options := sitetest.Options{
		Files: os.DirFS("templates"),
		Configure: func(config *webthings.Config) {
			config.Locales = siteLocales
			config.Translations = translations
			config.ImportMap = imports
		},
	}

	s := sitetest.StartWith(t, options, siteModules()...)

	if theme != "default" {
		s.Post("/admin/theme", "/admin/theme/activate", url.Values{"theme": {theme}})
	}

	for _, page := range []struct{ module, title, slug string }{
		{"news", "Nieuws", "nieuws"},
		{"blog", "Blog", "blog"},
		{"agenda", "Agenda", "agenda"},
		{"signup", "Aanmelden", "aanmelden"},
		{"comments", "Gastenboek", "gastenboek"},
		{"polls", "Peilingen", "peilingen"},
		{"newsletter", "Nieuwsbrief", "nieuwsbrief"},
	} {
		s.MountIn(page.module, sitetest.Translation{Locale: "nl", Title: page.title, Slug: page.slug})
	}

	for _, section := range []string{"news", "blog"} {
		post := s.Submit("/admin/"+section+"/new?locale=nl", url.Values{
			"Title": {"Havenfestival"},
			"Slug":  {"havenfestival"},
			"Date":  {time.Now().Format(time.DateOnly)},
			"Body":  {"Boten en muziek."},
		})
		s.Publish(post + "?locale=nl")
	}

	day := time.Now().AddDate(0, 1, 0).Format(time.DateOnly)
	event := s.Submit("/admin/agenda/new?locale=nl", url.Values{
		"Title":    {"Proeverij"},
		"Slug":     {"proeverij"},
		"Start":    {day + "T14:30"},
		"End":      {day + "T16:00"},
		"Timezone": {"Europe/Amsterdam"},
	})
	s.Publish(event + "?locale=nl")

	return s
}

// TestErrorPage breaks the page template: visitors get the theme's
// 500.gohtml, with status 500 and without the error, which is logged.
func TestErrorPage(t *testing.T) {
	files := fstest.MapFS{}

	err := fs.WalkDir(os.DirFS("templates"), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		data, err := fs.ReadFile(os.DirFS("templates"), name)
		files[name] = &fstest.MapFile{Data: data}

		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	files["default/page.gohtml"] = &fstest.MapFile{Data: []byte(`<main id="content">{{.Missing}}</main>`)}

	translations, err := frontend.Translations("translations")
	if err != nil {
		t.Fatal(err)
	}

	imports, err := importMap()
	if err != nil {
		t.Fatal(err)
	}

	// The error is logged, for the site's developers.
	var logged bytes.Buffer

	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	t.Cleanup(func() {
		slog.SetDefault(defaultLogger)
	})

	options := sitetest.Options{
		Files: files,
		Configure: func(config *webthings.Config) {
			config.Locales = siteLocales
			config.Translations = translations
			config.ImportMap = imports
		},
	}

	s := sitetest.StartWith(t, options, siteModules()...)

	status, body := s.Visitor().Get("/nl/")
	if status != http.StatusInternalServerError || !strings.Contains(body, "<h1>Er ging iets mis</h1>") ||
		strings.Contains(body, "Missing") {
		t.Fatalf("broken page: %d\n%s", status, body)
	}

	if !strings.Contains(logged.String(), "can't evaluate field Missing") {
		t.Fatalf("the error is not logged:\n%s", logged.String())
	}
}

// TestDevelopment starts the website as task dev does, with
// WEBTHINGS_DEVELOPMENT=true and the output cache as main leaves it: an edit
// to a template file shows on the next request without a restart, and a
// failing page shows the error. Without it, neither happens.
func TestDevelopment(t *testing.T) {
	t.Setenv("WEBTHINGS_DEVELOPMENT", "")

	if development(&webthings.Config{}) {
		t.Fatal("development mode without WEBTHINGS_DEVELOPMENT")
	}

	t.Setenv("WEBTHINGS_DEVELOPMENT", "true")

	dir := t.TempDir()

	err := os.CopyFS(dir, os.DirFS("templates"))
	if err != nil {
		t.Fatal(err)
	}

	translations, err := frontend.Translations("translations")
	if err != nil {
		t.Fatal(err)
	}

	imports, err := importMap()
	if err != nil {
		t.Fatal(err)
	}

	options := sitetest.Options{
		Files: os.DirFS(dir),
		Configure: func(config *webthings.Config) {
			config.Locales = siteLocales
			config.Translations = translations
			config.ImportMap = imports
			// main leaves the output cache to the site setting.
			config.OutputCacheTTL = 0
			development(config)
		},
	}

	s := sitetest.StartWith(t, options, siteModules()...)
	visitor := s.Visitor()
	page := filepath.Join(dir, "default", "page.gohtml")

	write := func(template string) {
		err := os.WriteFile(page, []byte(template), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	write(`<main id="content">first edit</main>`)
	visitor.Expect("/nl/", http.StatusOK, "first edit")

	write(`<main id="content">second edit</main>`)
	visitor.Expect("/nl/", http.StatusOK, "second edit")

	write(`<main id="content">{{.Missing}}</main>`)

	status, body := visitor.Get("/nl/")
	if status != http.StatusInternalServerError || !strings.Contains(body, "Missing") {
		t.Fatalf("broken page without the error: %d\n%s", status, body)
	}
}

// TestAdminCommand checks "go run . admin <email>": an administrator
// account and a link on the listening address to choose its password.
func TestAdminCommand(t *testing.T) {
	app := webthings.New(webthings.Config{
		Name:      "My website",
		StateDir:  t.TempDir(),
		Locales:   siteLocales,
		Templates: os.DirFS("templates"),
		Insecure:  true,
	})

	if err := app.Use(siteModules()...); err != nil {
		t.Fatal(err)
	}

	link, err := inviteAdministrator(t.Context(), app, "owner@example.test", ":8092")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(link, "http://localhost:8092/admin/reset-password?token=") {
		t.Fatalf("link %q", link)
	}
}
