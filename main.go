package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var mu = sync.RWMutex{}

type Film struct {
	Title    string
	Director string
}

var films []Film = []Film{
	{Title: "The Godfather", Director: "Francis Ford Coppola"},
	{Title: "Blade Runner", Director: "Ridley Scott"},
}

func main() {

	http.HandleFunc("/", handleIndex)

	http.HandleFunc("/add-film/", handleAddFilm)

	http.HandleFunc("/delete-film/{title}", handleDeleteFilm)

	fmt.Println("Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	templ := template.Must(template.ParseFiles("index.html"))
	mu.RLock()
	films := map[string][]Film{
		"Films": films,
	}
	mu.RUnlock()
	templ.Execute(w, films)
}

func handleAddFilm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Bad Request - Method '%s' not allowed for adding a film", r.Method)
		return
	}
	isHtmx := r.Header.Get("HX-Request")
	if isHtmx != "true" {
		http.Error(w, "Not an HTMX request", http.StatusBadRequest)
		log.Print("Bad Request - Not an HTMX request for adding a film")
		return
	}
	log.Print("HTMX requerst received for adding a film")
	title := r.PostFormValue("title")
	director := r.PostFormValue("director")

	if title == "" || director == "" {
		http.Error(w, "Title and Director are requruired", http.StatusBadRequest)

		log.Print("Bad Request - Title and Director are required for adding a film")
		return
	}

	film := Film{Title: title, Director: director}
	mu.Lock()
	time.Sleep(2 * time.Second)
	films = append(films, film)
	mu.Unlock()

	htmlStr := fmt.Sprintf(`
<li class="list-group-item bg-primary text-white">
    <p>%s - %s</p> 
    <button type="button" class="btn btn-danger" 
        hx-delete="/delete-film/%s"
        hx-swap="outerHTML"
        hx-target="#film-list"  
        hx-on:htmx:beforeRequest="document.getElementById('delete-spinner-%s').style.opacity='1'"
        hx-on:htmx:afterRequest="document.getElementById('delete-spinner-%s').style.opacity='0'">
        
        <span class="spinner-border spinner-border-sm htmx-indicator"
            id="delete-spinner-%s"
            role="status" aria-hidden="true"
            style="opacity:0; transition:opacity .2s;"></span>
        Delete
    </button>
</li>`, title, director, url.PathEscape(title), title, title, title)

	w.Header().Set("HX-Trigger", "filmAdded")
	w.Header().Set("HX-Trigger-After", "filmAdded")
	teml, err := template.New("film").Parse(htmlStr)
	if err != nil {
		http.Error(w, "Failed to create template", http.StatusInternalServerError)
		log.Fatal("Failed to create template:", err)
	}

	teml.Execute(w, nil)

}

func handleDeleteFilm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		log.Print("Bad Request - Method '%s' not allowed for deleting a film", r.Method)
		return
	}
	isHtmx := r.Header.Get("HX-Request")
	if isHtmx != "true" {
		http.Error(w, "Not an HTMX request", http.StatusBadRequest)
		log.Print("Bad Request - Not an HTMX request for deleting a film")
		return
	}
	log.Print("HTMX request received for deleting a film")

	enc := r.PathValue("title")
	title, err := url.PathUnescape(enc)
	if err != nil {
		http.Error(w, "Bad title", http.StatusBadRequest)
		return
	}

	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		log.Print("Bad Request - Title is required for deleting a film")
		return
	}

	mu.Lock()
	for i, film := range films {
		if film.Title == title {
			films = append(films[:i], films[i+1:]...)
			break
		}
	}
	fimlData := map[string][]Film{
		"Films": films,
	}
	mu.Unlock()
	templ := template.Must(template.ParseFiles("index.html"))
	w.Header().Set("HX-Trigger", "filmDeleted")
	err = templ.ExecuteTemplate(w, "film-list", fimlData)
	if err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		log.Fatal("Failed to execute template:", err)
	}
	log.Printf("Film '%s' deleted successfully", title)
}
