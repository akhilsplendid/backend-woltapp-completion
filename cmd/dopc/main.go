package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "time"

    "backend-woltapp-completion/internal/handler"
    "backend-woltapp-completion/internal/homeapi"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }

    baseURL := os.Getenv("HOME_ASSIGNMENT_API_BASE")
    if baseURL == "" {
        baseURL = "https://consumer-api.development.dev.woltapi.com"
    }

    httpClient := &http.Client{Timeout: 8 * time.Second}
    api := homeapi.New(baseURL, httpClient)

    mux := http.NewServeMux()
    mux.Handle("/api/v1/delivery-order-price", handler.PriceHandler(api))

    srv := &http.Server{
        Addr:              ":" + port,
        Handler:           mux,
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Printf("DOPC listening on :%s", port)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server error: %v", err)
    }

    _ = srv.Shutdown(context.Background())
}

