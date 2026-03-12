package main

func main() {
	server := newServer("localhost:8080", nil)

	server.HandleFunc("/hello", func(res Response, req *Request) {
		//res.sendString("Hello World!")
	})

	server.ListenAndServe()
}

