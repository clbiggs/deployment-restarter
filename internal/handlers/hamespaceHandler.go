package handlers

import (
	"context"
	"deployment-restarter/internal/middleware"
	"deployment-restarter/pkg/auth"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // alias for meta/v1
	"k8s.io/client-go/kubernetes"
	"net/http"
)

func GetNamespaceHandler(kubeClient *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r.Context())
		if !ok {
			http.Error(w, "Claims not found", http.StatusInternalServerError)
			return
		}

		var nsList *v1.NamespaceList
		var err error
		if claims.Role == auth.RoleAdmin {
			// admins get all namespaces
			nsList, err = kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			if err != nil {
				http.Error(w, fmt.Sprintf("Error fetching namespaces: %v", err), http.StatusInternalServerError)
				return
			}
		} else {
			// get namespaces with matching role label
			labelKey := fmt.Sprintf("ngic.com/restart.%s", claims.Role)
			labelSelector := fmt.Sprintf("%s=true", labelKey)
			nsList, err = kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("Error fetching namespaces: %v", err), http.StatusInternalServerError)
				return
			}
		}
		html := "<ul>"
		for _, ns := range nsList.Items {
			html += fmt.Sprintf(`<li><a hx-get="/api/namespaces/%s" hx-target="#deployments" hx-swap="innerHTML">%s</a></li>`, ns.Name, ns.Name)
		}
		html += "</ul>"
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}
}
