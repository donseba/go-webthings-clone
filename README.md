# WebThings starter

Copy this folder to start a website. You build the frontend; WebThings is the
backend: database, admin, pages, media, languages and modules.

```text
main.go         configuration and the list of modules
templates/      one folder per theme
  default/        the plain theme: layout, pages and one folder per module (Go templates)
  default/assets/ CSS, JavaScript and images, served at /theme/
  editorial/      a second, complete theme (see The editorial theme below)
importmap.json  the themes' JavaScript packages, vendored in assets/vendor/
assets.go       embeds assets/vendor/ and makes it the site's import map
translations/   the website's own UI texts per language (en.po, nl.po, ...)
var/            database and uploads, created on first start
```

## Run

```sh
task run
```

Open http://127.0.0.1:8092/ and the admin at http://127.0.0.1:8092/admin/.
The first start creates an admin account from `WEBTHINGS_ADMIN_EMAIL` and
`WEBTHINGS_ADMIN_PASSWORD` (the task sets local defaults; use your own for a real
site). After that the account is kept and these variables are ignored.
Started without them, the site logs a one-time setup link instead, where
you create the first administrator, and the admin then opens its setup
guide, Getting started (*Aan de slag* in the Dutch admin, the starter's
default).

```sh
go run . admin you@example.com
```

creates an administrator account and prints a link where its owner chooses
the password. The link works for one hour, and no password goes into the
environment. Use it when the setup link is lost or the site needs another
administrator; an address that has an account is refused.

`task run` serves the site as in production: templates are read once, and
the pages visitors get are kept in memory for a minute (the output cache,
Site settings → Performance). Changes made in the admin show at once, and a
browser signed in to the admin always gets fresh pages.

While you change the templates, run `task dev` instead. It serves on the
same address with `WEBTHINGS_DEVELOPMENT=true`: the templates are read again
from `templates/` on every request, so an edit shows on the next request
without a restart, a failing page shows the error with its template and
line, and the output cache is off. The site logs
`development mode (WEBTHINGS_DEVELOPMENT): …` when it starts. Never set it
in production: visitors would see the details of errors. Without it, a
broken template gives visitors the theme's `500.gohtml` in their language,
without the error, which goes to the log.

## Make it yours

- **Modules**: edit `app.Use(...)` in `main.go`. Each module gets an admin
  section; put it on the site by creating a page in the admin and choosing the
  module.
- **Design**: the templates in `templates/default/` are plain HTML that uses
  every available field, without JavaScript; build your design around them
  and style it in `templates/default/assets/site.css`.
  `templates/editorial/` is a finished design with HTMX navigation; see
  [The editorial theme](#the-editorial-theme).
  - `layout.gohtml` gets the page context (`frontend.View`: site, menu,
    languages, title); `page.gohtml` and `404.gohtml` get a `frontend.Page` and
    a `frontend.NotFound`. `500.gohtml` is the page visitors get when a
    template or a module fails: it is shown on its own, without the layout,
    and never shows the error, which goes to the log. Its `{{.Message}}` is
    a safe sentence and `{{.Title}}` the heading, such as "Internal Server
    Error", both already in the page's language, and `View` (`@model`)
    gives it the site, the theme and the language. Its `{{styles}}` links
    are versioned and carry their integrity as on other pages, also when a
    visitor's form fails.
  - The layout loads its stylesheets with
    `{{styles "webthings.css" "site.css"}}`: go-partial's
    `partial-errors.css` first, which core serves and which styles the box
    that takes the place of a part of a page that failed, then
    `webthings.css` and `site.css`, and last the theme's values stylesheet,
    which sets `--accent` to the colour chosen under Theme. Each link has
    the file's Subresource Integrity; `{{assetScript "site.js" "type"
    "module"}}` does the same for a script in `assets/`.
  - The layout places the tags of the page in `<head>` with
    `{{slot "head"}}`. The themes have no `regions/head.gohtml`, so
    WebThings writes them: the title, the description, the canonical and
    alternate links, the sharing tags (`og:` and `twitter:card`), the
    favicon from Site settings, the feed links, `{{apiLink}}` and the
    structured data. On a page in more than one language, the alternate
    links include the address in the default language as
    `hreflang="x-default"`. `{{apiLink}}` links the page in the content API
    while `Config.API` turns the API on; this site leaves it off, so it
    writes nothing. For tags of your own, add a `regions/head.gohtml` that
    starts with `{{template "webthings/head-tags.gohtml" .}}`, as
    `task templates -- site` writes it; one without that line replaces
    WebThings' tags.
  - `{{feedLinks}}`, one of those tags, links to the Atom and RSS feeds of
    the page shown, so browsers and feed readers find them: the newest posts
    of `news` and `blog` at `<page>/feed.atom` and `<page>/feed.rss`, also
    per category, and the upcoming events of `agenda`. A feed lists at most
    20 items unless Site settings → General → Items in feeds says otherwise.
  - `{{structuredData}}`, one of those tags, writes the page's
    schema.org JSON-LD for search engines: the site, the organisation from
    Site settings, the trail and the page, and an `Article` on a post of
    `news` or `blog` and an `Event` on an event of `agenda`, with an `Offer`
    while its registration in `signup` is open or full. The lists of
    `news`, `blog` and `agenda` carry an `ItemList` of their posts or events.
  - Module templates live in a folder named after the module's key, for
    example `templates/default/agenda/list.gohtml` for `events.Module("agenda", ...)`.
    Each gets the module's own data; the `@dot` line at the top names its type.
  - Instances that should look the same can share a folder named after the
    module type instead, for example `templates/default/publishing/` for news and blog.
  - After adding a module, write its starting points with
    `task templates -- <key>` (`site` writes the layout, with
    `{{slot "head"}}` and `{{styles}}`, a head region,
    `regions/head.gohtml`, that adds to WebThings' tags, the page, 404 and
    search templates and the share component, `site/share.gohtml`). Delete
    the ones you keep as they are: without a copy the
    module's default is used for components and e-mail.
    Existing files are never overwritten. A missing template is reported in
    the log at startup, with the `task templates -- <key>` that writes it.
    `task templates` without a key prints the keys it knows: `site` and the
    keys of your modules.
  - At startup the site also checks every template of both themes and logs
    each problem as `template problem`, with the file, the line and a hint,
    such as "did you mean" or the module a function comes from ("add it to
    app.Use or remove the call"). With `task dev` the site does not start
    until they are fixed; with `task run` it starts unless every page of
    the active theme would fail. A call that finds nothing, such as
    `{{posts "nieuws" 3}}` for a key the site does not have, a missing file
    in `{{assetStyle}}` or an unknown `{{theme}}` setting, is logged once
    with what there is, and with `task dev` the page shows the error.
  - `task test` starts the site on a test database with your modules,
    templates and translations, renders every kind of page in both themes,
    checks that none has a `style` attribute or a `<style>` element, and
    checks the editorial theme's head tags, HTMX fragments and forms
    (`main_test.go`), so a mistake in a template fails there instead of on
    the live site. Change its pages and checks when you change the modules
    or the markup it looks for.
- **Content anywhere**: every template can show any module's content, for
  example the latest news on the home page or next to the blog:
  `{{range posts "news" 3}}…{{end}}`, `{{range events "agenda" 5}}…{{end}}`,
  `{{with album "photos" "carousel"}}…{{end}}`, `{{range entries "faq"}}…{{end}}`,
  `{{range testimonials "reviews" 3}}…{{end}}` and `{{moduleURL "news"}}`.
  Shared snippets go in `partials/` and work in every template:
  `{{template "partials/card.gohtml" .}}`. `partials/when.gohtml` takes a
  `frontend.When` and shows when something takes place, in the page's
  language, an event over several days with a time as "date, time –
  date, time": `signup/registration.gohtml` calls it with
  `{{template "partials/when.gohtml" .Subject.When}}`, and the agenda's
  `list.gohtml` and `event.gohtml` with `.When`.
- **Comments, polls, signups**: the interactive modules come with
  components, ready-made HTML for any template: `{{commentThread "comments" .ID}}`
  below posts, `{{pollBox "polls" ""}}` (the newest poll) on the home page,
  `{{newsletterForm "newsletter"}}` in the footer, `{{registrationForm "signup" .ID}}`
  on events and `{{shareBar "" .Title}}`. To change one, copy it with
  `task templates -- <key>` (for example `comments/thread.gohtml`) and edit
  it; the copy wins. `webthings.css`, which the layout links with
  `{{styles}}`, lays them out; set `--wt-accent`, `--wt-radius` and
  `--wt-gap` in `site.css`, or remove it from `{{styles}}` and style the
  `wt-*` classes yourself. A form you write yourself
  takes its form from the module, as the guestbook in `comments/list.gohtml`
  does: `{{form_open .Form}}` writes the form tag with the spam trap, which
  `webthings.css` hides by its class `form-honeypot`, and the
  way back to the element whose id is the module's key, and
  `{{form_field .Form "name" nil}}` renders a field with its label and
  limits, with the classes of `webthings.css` unless `Config.FormTheme`
  names a go-form theme of your own. After a post with mistakes, such a
  field shows the visitor's value again, with its message.
- **Social media**: the profiles are filled in under Site settings and shown
  with `{{range .Site.Profiles}}` in the layout (`{{range profiles}}`
  anywhere).
- **Themes**: another folder next to `default/` is another theme, chosen in
  the admin under Theme (with a preview). A theme only needs what differs:
  templates and assets it lacks come from `default/`, so a dark variant can be
  a single `assets/site.css`. An optional `theme.json` gives it a title and
  settings editors can change, used as `{{.Theme.Values.accent}}` in the
  layout or `{{theme "accent"}}` anywhere. The default theme's accent is a
  colour picker, `#2563eb` until an editor chooses another, with
  `"css": "--accent"` in `theme.json`: core serves the colour as `--accent`
  in `/theme/default/values.css`, which `{{styles}}` links last, after
  `site.css`. So the accent needs no `style` attribute, which a
  Content-Security-Policy without `'unsafe-inline'`, such as the site's own
  (Site settings → Security), blocks.
  `WEBTHINGS_THEME=dark task templates -- <key>` writes starting points into
  another theme.
- **Languages**: the content languages are set in the admin (Site settings).
  Translate your own template texts in `translations/<language>.po`; see
  [Translations](#translations).
- **Database**: SQLite in `var/` by default; the SQLite driver is pure Go, so
  the site builds without cgo (`CGO_ENABLED=0`). Set `WEBTHINGS_DB_DRIVER` to
  `pgx` or `mysql` and `WEBTHINGS_DB_DSN` to use Postgres or MySQL.

## The editorial theme

`templates/editorial/` is a second, complete theme: a single column of text
under a newspaper masthead, in system fonts, light or dark as the visitor's
system prefers. It has its own layout, regions, fragments and module
screens and needs nothing from `default/`: its `publishing/` folder serves
both `news` and `blog`, `events/` the agenda and `registrations/` the
sign-ups.

Choose it in the admin under Theme: Preview, then Use this theme. Its
settings there are the accent colour, vermilion, ink, forest or plum, the
typeface of the text, serif or sans, and the width of the text column, a
number from 32 to 52 rem, 42 by default. The layout writes the first two
as `data-accent` and `data-text` on `<html>`, which `site.css` reads. The
width has `"css": "--measure"` in `theme.json`: core serves it as
`--measure: 42rem;` in `/theme/editorial/values.css`, which
`{{styles}}` links last, after `site.css`. So the theme writes no `style`
attribute or `<style>` element. It has no inline
script either, apart from the import map, whose hash the site's policy
carries, and the structured data, which browsers do not run, so it works
with the policy set to enforce. The accent stays a
choice of four rather than a colour picker: each is a designed pair of a
light and a dark shade whose contrast with the page's background passes
WCAG AA in both modes, which a free colour could not promise.
`WEBTHINGS_THEME=editorial task templates -- <key>` writes starting points
into it.

With HTMX, links and forms swap only `<main id="content">`, and the head
tags WebThings writes and the regions in `regions/` (editor, languages and
menu) follow out of band; the
menu's items carry their `ElementID`, so a navigation sends only the items
that change. Three fragments update on their own: `comments` below a post,
`newsletter`, the signup in the footer, and `results`, the live search,
which brings the regions along, so the tab's title and the language links
follow the query. Language and admin links load a whole page (`noBoost`).
After a navigation `site.js` moves the focus to the new page's `h1`. The
lists of `news`, `blog` and `agenda` link to their Atom and RSS feeds,
at the addresses `{{feedURL "atom"}}` and `{{feedURL "rss"}}` give: on a
category's page, the category's own feed. Without JavaScript every link
and form loads a whole page.

The theme loads htmx and go-partial's `partial-interactions` through the
site's import map. `importmap.json` lists htmx 4.0.0, which `task vendor`
downloads into `assets/vendor/` with its 0BSD licence and a lock file of
its integrity; `assets.go` embeds the folder and `main.go` passes it as
`Config.ImportMap`, so the site needs no network. Core adds go-partial's
browser clients to the map. The layout's `{{importmap}}` maps them,
`site.js` imports `htmx` and `partial-interactions`, and the body's
`clientAttrs` set `keepOnServerError` and `reloadOnError`: a server error
is not swapped into the page, a link that gets one, or no answer, loads
as a whole page, and a form that gets one shows the server's error page.
The default theme loads no JavaScript.

The theme's texts are in `translations/en.po` and `nl.po`, with those of
the default theme; `task translations` and `task translations:check` cover
both themes.

## Configuration

`main.go` reads these environment variables; change `main.go` for anything
else in `webthings.Config`.

| Variable | Default | Meaning |
| --- | --- | --- |
| `WEBTHINGS_LISTEN` | `127.0.0.1:8092` | Address the site listens on. |
| `WEBTHINGS_NAME` | `My website` | Initial site title and the admin title. |
| `WEBTHINGS_STATE_DIR` | `var` | Folder for the SQLite database and uploaded media. |
| `WEBTHINGS_DB_DRIVER` | `sqlite` | `sqlite` (`sqlite3` works too), `pgx` or `mysql`. |
| `WEBTHINGS_DB_DSN` | `site.sqlite` in the state folder | Database connection string; required for `pgx` and `mysql`. |
| `WEBTHINGS_ADMIN_LANGUAGE` | `nl` | Admin language, `en` or `nl`. |
| `WEBTHINGS_ADMIN_EMAIL`, `WEBTHINGS_ADMIN_PASSWORD` | | First admin account, created on the first start only. |
| `WEBTHINGS_INSECURE_HTTP` | | `true` allows session cookies over plain HTTP, for local development only (`task run` sets it). |
| `WEBTHINGS_DEVELOPMENT` | | `true` turns on development mode (`task dev` sets it): templates reload on every request, failing pages show the error, no output cache. Never in production. |
| `WEBTHINGS_BASE_URL` | | Public address, for example `https://example.com`, used in e-mail links, share links, feeds and `sitemap.xml`. Set it in production. |
| `WEBTHINGS_TRUSTED_PROXIES` | | Comma-separated IP addresses or CIDR ranges of your reverse proxies, for example `127.0.0.1,10.0.0.0/8`; their `X-Forwarded-For` names the visitor. |
| `WEBTHINGS_CONTENT_SECURITY_POLICY` | | `off`, `report-only` or `enforce` sets the Content-Security-Policy (`Config.ContentSecurityPolicy`); empty leaves it to Site settings → Security, off until an administrator changes it. |
| `WEBTHINGS_SMTP_HOST` | | Mail server. Without it e-mail is written to the log instead of sent. |
| `WEBTHINGS_SMTP_PORT` | `587` | Mail server port (STARTTLS when offered; port 465 is not supported). |
| `WEBTHINGS_SMTP_USERNAME`, `WEBTHINGS_SMTP_PASSWORD` | | Login for the mail server, when it needs one. |
| `WEBTHINGS_SMTP_FROM` | | Sender, for example `My website <site@example.com>`; required with `WEBTHINGS_SMTP_HOST`. |
| `WEBTHINGS_THEME` | `default` | Theme that `task templates` writes into. |

## Translations

Template texts are written in English with `tl`, for example
`{{tl localizer "Read more"}}` (`tn` for plurals, `ctl`/`ctn` with a context),
and translated in `translations/<language>.po`. `main.go` loads the
catalogues with `frontend.Translations("translations")` and sets them as
`Config.Translations`; core translates its own texts with them too, such as
the visitor forms' labels and messages and the names of months and days.
English and Dutch have those names and the date patterns built in, so
`en.po` and `nl.po` need not translate them and the check below does not ask
for them there; a translation in them replaces the built-in one.

1. `task translations` scans every template, including the modules' default
   templates and e-mail, adds the texts core and the modules' forms
   translate in Go and the calendar's names and patterns, and writes them to
   `translations/translations.pot`.
2. Open a language's `.po` file in [Poedit](https://poedit.net/), choose
   *Translation › Update from POT File…* and pick `translations/translations.pot`.
3. Translate the new texts and save. Restart the site to load them.

`task translations:check` lists the texts each language still misses and fails
when any are missing, so it can run in CI (`task translations:check -- --csv`
for a spreadsheet). Texts that are not written literally, such as
`{{tl localizer .Weekday.String}}`, are reported as warnings: name each possible
text in a literal `tl` call (for example in a `{{define}}` that is never used),
or updating from the POT file marks their translations as obsolete.

## Building

`task build` writes a static binary to `bin/website`:

```sh
CGO_ENABLED=0 go build -tags nodynamic -o bin/website .
```

Keep `-tags nodynamic` in your own build scripts and Dockerfiles, certainly
for Alpine or `scratch` images. Without it the WebP encoder of the image
variants loads a system libwebp when one is installed (through purego), which
makes the Linux binary dynamically linked against glibc's loader
(`/lib64/ld-linux-x86-64.so.2`), so it does not start on Alpine or `scratch`.
With the tag it always uses its built-in encoder. `file bin/website` should
say "statically linked" for a Linux build; `go version -m bin/website` shows
the tags it was built with.

## A change of go-webthings

`go.mod` requires a released go-webthings, so the site builds anywhere. To
try a change of go-webthings in your site before it is released, check
go-webthings out next to it and build both in a Go workspace; see
[Development](https://github.com/donseba/go-webthings#development) in
go-webthings' README.

After copying, change the module path in `go.mod`.
