package handlers

import (
	"context"
	"deployment-restarter/internal/middleware"
	"deployment-restarter/pkg/auth"
	"fmt"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // alias for meta/v1
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"time"
)

func GetDeploymentsHandler(kubeClient *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		namespace := vars["namespace"]

		claims, ok := middleware.GetClaims(r.Context())
		if !ok {
			http.Error(w, "Claims not found", http.StatusInternalServerError)
			return
		}

		if claims.Role != auth.RoleAdmin {
			// Ensure the namespace has the appropriate label.
			ns, err := kubeClient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get namespace: %v", err), http.StatusInternalServerError)
				return
			}
			labelKey := fmt.Sprintf("ngic.com/restart.%s", claims.Role)
			if value, exists := ns.Labels[labelKey]; !exists || value != "true" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// Fetch deployments from Kubernetes.
		deployments, err := kubeClient.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching deployments: %v", err), http.StatusInternalServerError)
			return
		}

		// Build HTML for the deployments.
		// Add a refresh button to update the deployment list.
		html := fmt.Sprintf("<h2>Deployments in %s</h2>", namespace)
		html += fmt.Sprintf(`<button hx-get="/api/namespaces/%s" hx-target="#deployments" hx-swap="innerHTML">Refresh Deployments</button>`, namespace)
		html += "<ul>"
		for _, d := range deployments.Items {
			desired := int32(0)
			if d.Spec.Replicas != nil {
				desired = *d.Spec.Replicas
			}
			status := fmt.Sprintf("%d/%d ready", d.Status.ReadyReplicas, desired)
			html += fmt.Sprintf(
				`<li>%s - %s <button hx-post="/api/namespaces/%s/deployments/%s/restart" hx-target="#deployments" hx-swap="innterHTML">Restart</button></li>`,
				d.Name, status, namespace, d.Name)
		}
		html += "</ul>"
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}
}
func RestartDeploymentHandler(kubeClient *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		namespace := vars["namespace"]
		deploymentName := vars["deployment"]

		claims, ok := middleware.GetClaims(r.Context())
		if !ok {
			http.Error(w, "Claims not found", http.StatusInternalServerError)
			return
		}

		if claims.Role != auth.RoleAdmin {
			// Ensure the namespace has the appropriate label.
			ns, err := kubeClient.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get namespace: %v", err), http.StatusInternalServerError)
				return
			}
			labelKey := fmt.Sprintf("ngic.com/restart.%s", claims.Role)
			if value, exists := ns.Labels[labelKey]; !exists || value != "true" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// Patch the deployment's pod template annotation to trigger a rollout restart.
		patchData := fmt.Sprintf(`{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`,
			time.Now().Format(time.RFC3339))
		_, err := kubeClient.AppsV1().Deployments(namespace).Patch(context.Background(), deploymentName, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to restart deployment: %v", err), http.StatusInternalServerError)
			return
		}
		// Return the updated deployments.
		GetDeploymentsHandler(kubeClient)(w, r)
	}
}
