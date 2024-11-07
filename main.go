package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	fmt.Println("Server is starting at port 8080")
	mux.HandleFunc("/", handleRoot)

	http.ListenAndServe(":8080", mux)

}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}
