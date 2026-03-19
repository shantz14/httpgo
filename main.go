package main

func main() {
	server := newServer("localhost:8080", nil)

	server.HandleFunc("/hello", func(res Response, req *Request) {
		res.sendText("Hello World!")
	})

	server.ListenAndServe()
}

