// This is your website. Change the configuration and module list here, and
// the design in templates/<theme>/. WebThings runs everything else:
// database, admin, pages, media, languages and the selected modules.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	webthings "github.com/donseba/go-webthings"
	"github.com/donseba/go-webthings/frontend"
	"github.com/donseba/go-webthings/locales"
	"github.com/donseba/go-webthings/mail"
	"github.com/donseba/go-webthings/modules/comments"
	"github.com/donseba/go-webthings/modules/events"
	"github.com/donseba/go-webthings/modules/newsletter"
	"github.com/donseba/go-webthings/modules/polls"
	"github.com/donseba/go-webthings/modules/publishing"
	"github.com/donseba/go-webthings/modules/redirects"
	"github.com/donseba/go-webthings/modules/registrations"
	"github.com/donseba/go-webthings/modules/stats"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	translations, err := frontend.Translations("translations")
	if err != nil {
		log.Fatal(err)
	}

	sender, err := smtpSender()
	if err != nil {
		log.Fatal(err)
	}

	imports, err := importMap()
	if err != nil {
		log.Fatal(err)
	}

	config := webthings.Config{
		Name:          env("WEBTHINGS_NAME", "My website"),
		StateDir:      env("WEBTHINGS_STATE_DIR", "var"),
		Driver:        env("WEBTHINGS_DB_DRIVER", "sqlite"),
		DSN:           os.Getenv("WEBTHINGS_DB_DSN"),
		Locales:       siteLocales,
		AdminLanguage: env("WEBTHINGS_ADMIN_LANGUAGE", "nl"),
		AdminEmail:    os.Getenv("WEBTHINGS_ADMIN_EMAIL"),
		AdminPassword: os.Getenv("WEBTHINGS_ADMIN_PASSWORD"),
		Insecure:      os.Getenv("WEBTHINGS_INSECURE_HTTP") == "true",
		// Behind a reverse proxy, set the public address and the proxy's
		// address: links in e-mail and sitemap.xml then use the public
		// address, and visitors are known by their own IP address.
		BaseURL:        os.Getenv("WEBTHINGS_BASE_URL"),
		TrustedProxies: list(os.Getenv("WEBTHINGS_TRUSTED_PROXIES")),
		Mail:           sender,
		Templates:      os.DirFS("templates"),
		// The themes' JavaScript packages; see importMap.
		ImportMap:    imports,
		Translations: translations,
		// off, report-only or enforce decides the Content-Security-Policy
		// here; empty leaves it to the admin's Security settings.
		ContentSecurityPolicy: os.Getenv("WEBTHINGS_CONTENT_SECURITY_POLICY"),
	}

	developing := development(&config)
	app := webthings.New(config)

	err = app.Use(siteModules()...)
	if err != nil {
		log.Fatal(err)
	}

	// "go run . templates agenda" writes starting points for a module's
	// templates ("site" for the layout, page, 404 and search templates and
	// the share component in site/share.gohtml) into the theme in
	// WEBTHINGS_THEME, "default" when unset. Existing files are kept.
	if len(os.Args) > 1 && os.Args[1] == "templates" {
		if len(os.Args) == 2 {
			log.Fatalf("usage: go run . templates <key>...; keys: %s", strings.Join(app.TemplateKeys(), ", "))
		}

		theme := filepath.Join("templates", env("WEBTHINGS_THEME", "default"))

		for _, key := range os.Args[2:] {
			written, err := app.WriteTemplates(theme, key)
			if err != nil {
				log.Fatal(err)
			}

			for _, file := range written {
				log.Println("wrote", file)
			}
		}

		return
	}

	// "go run . translations" writes the texts of the templates to
	// translations/translations.pot; "go run . translations check" lists the
	// texts each language's .po file still misses (add --csv for a CSV).
	if len(os.Args) > 1 && os.Args[1] == "translations" {
		if err := app.Translations("translations", os.Args[2:]); err != nil {
			log.Fatal(err)
		}

		return
	}

	// "go run . admin someone@example.com" creates an administrator account
	// and prints a link with which its owner chooses a password, for a site
	// without the setup link at hand or that needs another administrator.
	// The link works for one hour.
	if len(os.Args) > 1 && os.Args[1] == "admin" {
		if len(os.Args) != 3 {
			log.Fatal("usage: go run . admin <email>")
		}

		link, err := inviteAdministrator(ctx, app, os.Args[2], env("WEBTHINGS_LISTEN", "127.0.0.1:8092"))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Open this link within an hour to choose the password of", os.Args[2]+":")
		fmt.Println(link)

		return
	}

	if developing {
		log.Println("development mode (WEBTHINGS_DEVELOPMENT): templates reload from disk, detailed errors, no page cache")
	}

	if err := app.ListenAndServe(ctx, env("WEBTHINGS_LISTEN", "127.0.0.1:8092")); err != nil {
		log.Fatal(err)
	}
}

// siteLocales are the website's languages; the first is the default.
var siteLocales = locales.Config{Default: "nl", Enabled: []string{"nl", "en"}}

// siteModules are the modules of the website; main_test.go starts the same.
//
// Group places a module's admin sections under a heading of the admin
// navigation; modules without one are listed under "Modules". Agenda,
// Content, Visitors and Website are translated by the admin.
func siteModules() []webthings.Module {
	return []webthings.Module{
		publishing.Module("news", "News").Group("Content"),
		publishing.Module("blog", "Blog").Group("Content"),
		events.Module("agenda", "Agenda").Group("Agenda"),
		registrations.Module("signup", "Sign-ups", registrations.ForEvents("agenda")).Group("Agenda"),

		// Visitors take part: comments below posts, a poll on the home page
		// and a newsletter signup in the footer. The templates show them with
		// the modules' components, such as {{commentThread "comments" .ID}}.
		comments.Module("comments", "Comments").Group("Visitors"),
		polls.Module("polls", "Polls").Group("Visitors"),
		newsletter.Module("newsletter", "Newsletter").Group("Visitors"),

		// Site features need no key.
		redirects.Module().Group("Website"),
		stats.Module().Group("Website"),
	}
}

// development turns on the development mode when WEBTHINGS_DEVELOPMENT is
// "true", as task dev sets it: the templates are read again from the
// templates folder on every request, so an edit shows on the next request
// without a restart; a failing page shows the detailed error page with the
// error and its template; and the output cache is off, so no page is served
// from before the edit. It is off by default: visitors must never see the
// errors' details, and main logs a line when it is on.
func development(config *webthings.Config) bool {
	if os.Getenv("WEBTHINGS_DEVELOPMENT") != "true" {
		return false
	}

	config.ReloadTemplates = true
	config.Development = true

	return true
}

// inviteAdministrator starts app, creates an administrator for email with
// App.InviteAdministrator and returns the link to choose its password. A
// site without BaseURL gets the link on listen, the address it serves on.
func inviteAdministrator(ctx context.Context, app *webthings.App, email, listen string) (string, error) {
	if err := app.Start(ctx); err != nil {
		return "", err
	}

	defer func() { _ = app.Close() }()

	link, err := app.InviteAdministrator(ctx, email)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(link, "/") {
		return link, nil
	}

	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return link, nil
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, port) + link, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// list splits a comma-separated value and drops empty entries.
func list(value string) []string {
	var items []string

	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}

// smtpSender sends e-mail through the server in WEBTHINGS_SMTP_HOST. Without
// a host it returns nil, so messages are written to the log instead.
func smtpSender() (mail.Sender, error) {
	host := os.Getenv("WEBTHINGS_SMTP_HOST")
	if host == "" {
		return nil, nil
	}

	from := os.Getenv("WEBTHINGS_SMTP_FROM")
	if from == "" {
		return nil, errors.New("WEBTHINGS_SMTP_FROM is required with WEBTHINGS_SMTP_HOST")
	}

	port := 587

	if value := os.Getenv("WEBTHINGS_SMTP_PORT"); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("WEBTHINGS_SMTP_PORT: %w", err)
		}

		port = number
	}

	return mail.SMTP{
		Host:     host,
		Port:     port,
		Username: os.Getenv("WEBTHINGS_SMTP_USERNAME"),
		Password: os.Getenv("WEBTHINGS_SMTP_PASSWORD"),
		From:     from,
	}, nil
}
