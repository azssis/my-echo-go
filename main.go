package main

import (
	"log"
	"my-echo-go/pkg/handler"
	"my-echo-go/pkg/middleware"
	"net/http"
	"os"
)

var (
	Version   = "-"
	GitCommit = "-"
)

func main() {
	log.Printf("my-echo-go: Version: %s / Commit: %s", Version, GitCommit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler.InitOption()
	middleware.InitOption()

	mux := http.NewServeMux()
	mux.Handle("/", middleware.LoggingMiddleware(http.HandlerFunc(handler.Echo)))

	certPath, isCertExist := os.LookupEnv("CERT_PATH")
	if certPath == "" {
		certPath = "tls/server.crt"
	}
	privateKeyPath, isPrivateKeyExist := os.LookupEnv("PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		privateKeyPath = "tls/server.key"
	}

	if isCertExist && isPrivateKeyExist {
		go func() {
			if _, err := os.Stat(certPath); os.IsNotExist(err) {
				log.Println(err)
				return
			}
			if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
				log.Println(err)
				return
			}
			log.Println("Listening TLS enabled on" + port)
			log.Fatal(http.ListenAndServeTLS(":443", certPath, privateKeyPath, mux))
		}()
		log.Fatal(http.ListenAndServe(":8080", mux))
	} else {
		log.Println("Listening on " + port)
		log.Fatal(http.ListenAndServe(":"+port, mux))
	}
}
