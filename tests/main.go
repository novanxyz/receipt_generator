package main

import (
	"log"
	"os"
	"runtime"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	_ "goto.io/tax_receipt"
)

func main() {

	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
	runtime.GC()
}
