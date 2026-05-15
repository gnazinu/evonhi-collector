package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// 1. Construir la configuración a partir del archivo kubeconfig local
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// 2. Crear el cliente de Kubernetes
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %s", err.Error())
	}

	ctx := context.Background()

	// 3. Extraer ServiceAccounts
	fmt.Println("--- Fetching ServiceAccounts ---")
	saList, err := clientset.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing ServiceAccounts: %s", err.Error())
	}
	for _, sa := range saList.Items {
		fmt.Printf("Namespace: %s | Name: %s\n", sa.Namespace, sa.Name)
	}

	// 4. Extraer RoleBindings
	fmt.Println("\n--- Fetching RoleBindings ---")
	rbList, err := clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing RoleBindings: %s", err.Error())
	}
	for _, rb := range rbList.Items {
		fmt.Printf("Namespace: %s | Name: %s | RoleRef: %s\n", rb.Namespace, rb.Name, rb.RoleRef.Name)
	}
}