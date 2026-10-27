// The editorial theme's script. The site works without it: links then load
// whole pages and forms return to their anchor through the back field
// form_open writes. The layout's {{importmap}} maps the names it imports:
// htmx, vendored in the site's assets/vendor (importmap.json), which starts
// itself, and go-partial's interactions client, which core adds.
import "htmx";
// A link that gets a 5xx answer, or none, loads as a whole page, and a form
// shows the server's error page (the body's clientAttrs "reloadOnError"), so
// the visitor sees what went wrong rather than the error swapped into the
// page.
import "partial-interactions";

// A navigation swaps <main id="content">. Screen readers and keyboards then
// start from the new content: its heading takes the focus, without a scroll.
let content = document.getElementById("content");

document.addEventListener("htmx:after:settle", () => {
    const current = document.getElementById("content");
    if (!current || current === content) {
        return;
    }

    content = current;

    const heading = current.querySelector("h1");
    if (!heading) {
        return;
    }

    heading.setAttribute("tabindex", "-1");
    heading.focus({ preventScroll: true });
});
