package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"github.com/satori/go.uuid"
	"html/template"
)

var tpl *template.Template

var clients = make(map[*websocket.Conn]bool) // clients connected to ws
var sessions = map[string]string{}           // list of open sessions;;;; map[sessionId] -> user email
var users = map[string]user{}                // list of all users ;;;; map[user email] -> user

var broadcast = make(chan Message)

func init() {
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
}

type user struct {
	Email    string
	UserName string
	Password []byte
}

type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	http.HandleFunc("/ws", handleConnections)

	http.HandleFunc("/", index)
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/signup", signup)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)

	log.Println("Starting server")
	//err := http.ListenAndServe(":80", nil)
	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/black.zone/fullchain.pem", "/etc/letsencrypt/live/black.zone/privkey.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func index(w http.ResponseWriter, req *http.Request) {
	u := getUser(w, req)
	go handleMessages()
	log.Println(u.UserName)
	tpl.ExecuteTemplate(w, "index.gohtml", u)
}

func connect(w http.ResponseWriter, req *http.Request) {
	u := getUser(w, req)
	go handleMessages()
	log.Println(u.UserName)
	tpl.ExecuteTemplate(w, "index.gohtml", u)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true
	u := getUser(w, r)

	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object

		err := ws.ReadJSON(&msg)
		msg.Email = u.Email
		msg.Username = u.UserName
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

// TODO remove client from list?
func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func signup(w http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	var u user
	// process form submission
	if req.Method == http.MethodPost {
		// get form values
		username := req.FormValue("username")
		password := req.FormValue("password")
		email := req.FormValue("email")

		// email taken?
		if _, ok := users[email]; ok {
			// TODO email taken message?
			http.Redirect(w, req, "/", http.StatusSeeOther)
			return
		}
		// create session
		sID := uuid.NewV4()
		c := &http.Cookie{
			Name:  "session",
			Value: sID.String(),
		}
		http.SetCookie(w, c)
		sessions[c.Value] = email
		// store user in dbUsers
		bs, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		u = user{email, username, bs}
		users[email] = u

		// redirect
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	tpl.ExecuteTemplate(w, "signup.gohtml", u)
}

func login(w http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	var u user
	// process form submission
	if req.Method == http.MethodPost {
		email := req.FormValue("email")
		p := req.FormValue("password")

		// is there a username?
		u, ok := users[email]
		if !ok {
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}
		// does the entered password match the stored password?
		err := bcrypt.CompareHashAndPassword(u.Password, []byte(p))
		if err != nil {
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}
		// create session
		sID := uuid.NewV4()
		c := &http.Cookie{
			Name:  "session",
			Value: sID.String(),
		}
		http.SetCookie(w, c)
		sessions[c.Value] = email
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	tpl.ExecuteTemplate(w, "login.gohtml", u)
}

func logout(w http.ResponseWriter, req *http.Request) {
	if !alreadyLoggedIn(req) {
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	c, _ := req.Cookie("session")
	// delete the session
	delete(sessions, c.Value)
	// remove the cookie
	emptyCookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
	}
	http.SetCookie(w, emptyCookie)
	http.Redirect(w, req, "/login", http.StatusSeeOther)
}

func getUser(w http.ResponseWriter, req *http.Request) user {
	// get cookie
	c, err := req.Cookie("session")
	if err != nil {
		sID := uuid.NewV4()
		c = &http.Cookie{
			Name:  "session",
			Value: sID.String(),
		}

	}
	http.SetCookie(w, c)

	// if the user exists already, get user
	var u user
	if email, ok := sessions[c.Value]; ok {
		u = users[email]
	}
	return u
}

func alreadyLoggedIn(req *http.Request) bool {
	c, err := req.Cookie("session")
	if err != nil {
		return false
	}
	email := sessions[c.Value]
	_, ok := users[email]
	return ok
}
