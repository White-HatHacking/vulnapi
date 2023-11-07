package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"-"`
}

var userMap = make(map[string]User)
var mu sync.RWMutex

var (
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func main() {
	router := mux.NewRouter() // This object is responsible for routing users that access different pages to different content

	// Route GET requests to static content
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	router.PathPrefix("/").HandlerFunc(indexFileHandler("static/"))

	// Route POST requests to functions
	router.HandleFunc("/admin/", adminHandler)
	router.HandleFunc("/register/", registerHandler).Methods("POST")
	router.HandleFunc("/login/", loginHandler).Methods("POST")

	http.Handle("/", router)

	port := 8080
	fmt.Printf("Server is running on port %d\n", port)
	//log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func indexFileHandler(rootDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {

		// Get the absolute path to prevent directory traversal
		path, err := filepath.Abs(request.URL.Path)
		if err != nil {
			// If we can't get the absolute path, return a not found error
			http.NotFound(w, request)
			return
		}
		if strings.HasPrefix(path, filepath.Clean(rootDir)) {
			http.ServeFile(w, request, filepath.Join(rootDir, request.URL.Path, "index.html"))
		} else {
			http.ServeFile(w, request, filepath.Join(rootDir, request.URL.Path)) // serve the file as is
		}
	}
}

func adminHandler(writer http.ResponseWriter, request *http.Request) {
	http.Error(writer, "Access denied", http.StatusForbidden)
	//http.ServeFile(writer, request, "static/admin/index.html")
}

func registerHandler(writer http.ResponseWriter, request *http.Request) {
	var user User
	err := json.NewDecoder(request.Body).Decode(&user)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := userMap[user.Username]; exists {
		http.Error(writer, "User already exists", http.StatusConflict)
		return
	}

	userMap[user.Username] = user
	writer.WriteHeader(http.StatusCreated)
	fmt.Println("User created: \"", user, "\"")
}

func loginHandler(writer http.ResponseWriter, request *http.Request) {
	var user User
	session, err := store.Get(request, "session-name")
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(request.Body).Decode(&user)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	storedUser, exists := userMap[user.Username]
	if !exists || storedUser.Password != user.Password {
		http.Error(writer, "Invalid username or password", http.StatusUnauthorized)
		fmt.Println("User failed to login: \"", user, "\"")
		return
	}

	if user.Username == "admin" {
		session.Values["authenticated"] = true
		// onBolaExploited()
		if err := session.Save(request, writer); err != nil {
			// Handle errors saving the session
			log.Printf("Error saving session: %v", err)
			http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	fmt.Println("User logged in: \"", user, "\"")
}
