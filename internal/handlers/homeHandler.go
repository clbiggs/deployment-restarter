package handlers

import (
	"html/template"
	"net/http"
)

// HomeHandler serves the main HTML page which uses htmx.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// If the user is not logged in, show a login link.
	var content string
	if _, err := r.Cookie("token"); err != nil {
		content = `<h1>Deployment Manager</h1>
<p><a href="/login">Login via Keycloak</a></p>`
	} else {
		content = `<h1>Deployment Manager</h1>
<!-- Namespaces will be loaded here via htmx -->
<div id="namespaces" hx-get="/api/namespaces" hx-trigger="load">
	Loading namespaces...
</div>
<!-- Deployments for the selected namespace will be shown here -->
<div id="deployments"></div>`
	}
	tmpl := template.Must(template.New("home").Parse(`
<!DOCTYPE html>
<html>
<head>
	<title>Deployment Manager</title>
	<script src="https://unpkg.com/htmx.org@1.9.2"></script>
</head>
<body>
	{{.}}
	<hr>
	<p><a href="/login">Re-login via Keycloak</a></p>
</body>
</html>
`))
	w.Header().Set("Content-Type", "text/html")
	_ = tmpl.Execute(w, template.HTML(content))
}
