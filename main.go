package main

func main() {
	server := newServer("localhost:8080", nil)
	server.ListenAndServe()
}

