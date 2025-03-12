package main

import (
	"deployment-restarter/internal/handlers"
	"deployment-restarter/internal/middleware"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"log"
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Settings struct {
	KubeconfigPath string `json:"kubeconfigPath"`
	ServerAddr     string `json:"serverAddr" default:":8080"`
}

func main() {
	settings, _ := getSettings()
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig when not running in-cluster.
		kubeconfig := settings.KubeconfigPath
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	serverVersion, err := kubeClient.ServerVersion()
	log.Printf("Server Version: %s", serverVersion.String())

	// Set up router using gorilla/mux.
	r := mux.NewRouter()

	// OAuth login endpoints.
	r.HandleFunc("/login", handlers.LoginHandler).Methods("GET")
	r.HandleFunc("/callback", handlers.CallbackHandler).Methods("GET")

	// Main page.
	r.HandleFunc("/", handlers.HomeHandler)

	// Protected API endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.JWTMiddleware)
	api.HandleFunc("/namespaces", handlers.GetNamespaceHandler(kubeClient)).Methods("GET")

	// Serve static files if needed.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("Server started at %s\n", settings.ServerAddr)
	log.Fatal(http.ListenAndServe(settings.ServerAddr, r))
}

func getSettings() (*Settings, error) {
	var s Settings
	err := envconfig.Process("deprestart", &s)
	return &s, err
}
