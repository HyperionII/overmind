package main

import (
	"log"
	"net/http"
	"text/template"
)

const (
	port = "2222"
)

var homeTemplate = template.Must(template.ParseFiles("static/index.html"))

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTemplate.Execute(w, nil)
}

func main() {
	log.Printf("Starting server at port %s...\n", port)

	s := NewServer()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", s.OnConnect)

	log.Println("Listening...")
	go s.Listen()
	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		log.Fatalln("main: listen and serve:", err)
	}
}
