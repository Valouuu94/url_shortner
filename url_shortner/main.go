package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var urlsCollection *mongo.Collection

type URLDocument struct {
	ShortKey    string `bson:"short_key"`
	OriginalURL string `bson:"original_url"`
}

func main() {
	initMongoDB()
	http.HandleFunc("/", handleForm)
	http.HandleFunc("/shorten", handleShorten)
	http.HandleFunc("/short/", handleRedirect)

	fmt.Println("URL Shortener is running on :3030")
	if err := http.ListenAndServe(":3030", nil); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initMongoDB() {
	var err error
	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	urlsCollection = client.Database("urlshortener").Collection("urls")
	fmt.Println("Connected to MongoDB")
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving the form")
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/shorten", http.StatusSeeOther)
		return
	}

	// Serve the HTML form
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>URL Shortener</title>
		</head>
		<body>
			<h2>URL Shortener</h2>
			<form method="post" action="/shorten">
				<input type="url" name="url" placeholder="Enter a URL" required>
				<input type="submit" value="Shorten">
			</form>
		</body>
		</html>
	`)
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	originalURL := r.FormValue("url")
	if originalURL == "" {
		http.Error(w, "URL parameter is missing", http.StatusBadRequest)
		return
	}

	shortKey := generateShortKey()

	_, err := urlsCollection.InsertOne(context.TODO(), URLDocument{
		ShortKey:    shortKey,
		OriginalURL: originalURL,
	})
	if err != nil {
		http.Error(w, "Failed to insert URL into DB", http.StatusInternalServerError)
		return
	}

	shortenedURL := fmt.Sprintf("http://localhost:3030/short/%s", shortKey)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<p>Shortened URL: <a href="%s">%s</a></p>`, shortenedURL, shortenedURL)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	shortKey := strings.TrimPrefix(r.URL.Path, "/short/")
	var doc URLDocument
	if err := urlsCollection.FindOne(context.TODO(), bson.M{"short_key": shortKey}).Decode(&doc); err != nil {
		http.Error(w, "Shortened key not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, doc.OriginalURL, http.StatusMovedPermanently)
}

func generateShortKey() string {
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
