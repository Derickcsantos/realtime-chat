package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/Derickcsantos/realtime-chat/internal/hub"
	"github.com/Derickcsantos/realtime-chat/internal/store"
	"github.com/Derickcsantos/realtime-chat/internal/api"
	"github.com/Derickcsantos/realtime-chat/internal/repo"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
	    port = "8080" // fallback para rodar local
	}
	addr := ":" + port


	if err := store.InitStore("mongodb+srv://Derick:Basquete@cluster0.dfboxft.mongodb.net/realtime-chat"); err != nil {
		log.Fatalf("Erro ao conectar ao MongoDB: %v", err)
	}
	defer store.CloseStore()

	h := hub.NewHub()
	go h.Run() // Você precisa implementar o método Run() no seu Hub

	r := mux.NewRouter()
    r.Handle("/", http.FileServer(http.Dir("web")))
    r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        hub.ServeWS(h, w, r)
    })

	r.HandleFunc("/api/hot", api.GetHotMessagesHandler).Methods("GET")
    r.HandleFunc("/api/cold", api.GetColdMessagesHandler).Methods("GET")
    r.HandleFunc("/api/message", api.PostMessageHandler).Methods("POST")
    r.HandleFunc("/api/flush", api.FlushHandler).Methods("POST")

	srv := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Servidor iniciado em http://localhost%v", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro ao iniciar o servidor: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := repo.FlushHotToCold(); err != nil {
				log.Println("Erro ao flush do Redis para Mongo:", err)
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Servidor desligando...")
	_ = srv.Close()
	// h.Close() pode ser implementado se quiser encerrar o Hub
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})

}
