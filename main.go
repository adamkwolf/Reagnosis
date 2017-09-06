package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"github.com/satori/go.uuid"
	"html/template"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"strings"
)

var tpl *template.Template

var clients = make(map[*websocket.Conn]bool) // clients connected to ws
var sessions = map[string]string{}           // list of open sessions;;;; map[sessionId] -> user email
var users = map[string]user{}                // list of all users ;;;; map[user email] -> uses

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

var db *sql.DB
var err error

func main() {

	db, err = sql.Open("mysql", "awsuser:mypassword@tcp(mydbinstance.cpnfukl3epsb.eu-central-1.rds.amazonaws.com:3306)/mydb?charset=utf8")
	if err != nil {
		log.Panicln("could not open database connection", err)
	}

	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Panicln("Could not creach database", err)
	}

	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/", index)
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/signup", signup)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)

	log.Println("Starting server")
	//err := http.ListenAndServe(":8080", nil)
	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/black.zone/fullchain.pem", "/etc/letsencrypt/live/black.zone/privkey.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func index(w http.ResponseWriter, req *http.Request) {
	if !alreadyLoggedIn(req) {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}

	u, err := getUser(w, req)
	if err != nil {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	go handleMessages()
	//log.Println(u.UserName)
	tpl.ExecuteTemplate(w, "index.gohtml", u)
}

func connect(w http.ResponseWriter, req *http.Request) {
	u, err := getUser(w, req)
	if err != nil {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
		return
	}
	go handleMessages()
	//log.Println(u.UserName)
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
	u, err := getUser(w, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

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

		query, err := db.Query(`SELECT * FROM users WHERE email=?;`, email)
		if err != nil {
			// user exists
			http.Redirect(w, req, "/signup", http.StatusSeeOther)
			return
		}

		var exists int
		for query.Next() {
			err := query.Scan(&exists)
			if err != nil {
				http.Redirect(w, req, "/signup", http.StatusSeeOther)
				return
			}
		}

		if exists >= 1 {
			http.Redirect(w, req, "/signup", http.StatusSeeOther)
			return
		}

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
			http.Redirect(w, req, "/signup", http.StatusSeeOther)
			return
		}

		stmt, err := db.Prepare(`INSERT INTO users VALUES (?, ?, ?, ?);`)
		if err != nil {
			http.Redirect(w, req, "/signup", http.StatusSeeOther)
			return
		}
		_, err = stmt.Exec(username, email, string(bs[:]), c.Value)
		if err != nil {
			http.Redirect(w, req, "/signup", http.StatusSeeOther)
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
		log.Println("already logged in")
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}
	var u user
	// process form submission
	if req.Method == http.MethodPost {
		email := req.FormValue("email")
		p := req.FormValue("password")

		// is there a username?
		//u, ok := users[email]
		//if !ok {
		//	log.Println("user doesnt exist")
		//	http.Redirect(w, req, "/login", http.StatusSeeOther)
		//	return
		//}
		// does the entered password match the stored password?

		query, error := db.Query(`SELECT pass FROM users WHERE UPPER(email)=?`, strings.ToUpper(email))
		if error != nil {
			http.Redirect(w, req, "/login", http.StatusSeeOther)
		}
		var retPass string
		for query.Next() {
			err := query.Scan(&retPass)
			if err != nil {
				http.Redirect(w, req, "/login", http.StatusSeeOther)
			}
		}

		err := bcrypt.CompareHashAndPassword([]byte(retPass), []byte(p))
		if err != nil {
			log.Println("wrong password")
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}

		// create session
		c := NewSessionId()

		http.SetCookie(w, c)

		stmt, err := db.Prepare(`UPDATE users SET sessionId=? WHERE email=?`)
		if err != nil {
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}
		_, err = stmt.Exec(c.Value, email)
		if err != nil {
			http.Redirect(w, req, "/login", http.StatusSeeOther)
			return
		}

		sessions[c.Value] = email
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	tpl.ExecuteTemplate(w, "login.gohtml", u)
}
func NewSessionId() *http.Cookie {
	sID := uuid.NewV4()
	c := &http.Cookie{
		Name:  "session",
		Value: sID.String(),
	}
	return c
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

func getUser(w http.ResponseWriter, req *http.Request) (user, error) {
	// get cookie
	c, err := req.Cookie("session")
	if err != nil || c == nil {
		http.Redirect(w, req, "/logout",http.StatusSeeOther)
		return user{}, err
	}

	http.SetCookie(w, c)
	query, error := db.Query(`SELECT * FROM users u WHERE u.sessionId=?`, c.Value)
	if error != nil {
		http.Redirect(w, req, "/login",http.StatusSeeOther)
		return user{}, err
	}

	var u user
	for query.Next() {
		var r user
		err := query.Scan(&r.UserName, &r.Email, &r.Password, &c.Value)
		if err != nil {
			http.Redirect(w, req, "/login",http.StatusSeeOther)
		}
		u = r
	}
	return u, nil
}

func alreadyLoggedIn(req *http.Request) bool {
	c, err := req.Cookie("session")
	if err != nil {
		return false
	}
	email := sessions[c.Value]

	query, error := db.Query(`SELECT COUNT(*) FROM users WHERE email=?`, email)
	if error != nil {
		return false
	}

	var rows int
	for query.Next() {
		err := query.Scan(&rows)
		if err != nil { return false }
	}

	return rows == 1
}
