package main

import (
	"log"
	"net/http"
	"text/template"
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
	log.Println("Starting server at port 2222...")

	s := NewServer()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/echo", s.OnConnect)

	log.Println("Listening...")
	go s.Listen()
	err := http.ListenAndServe(":2222", nil)

	if err != nil {
		log.Fatalln("main: listen and serve:", err)
	}
}
