package hello

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"

	"google.golang.org/appengine"
	//"appengine/log"
	//"appengine/user"

	//"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/drive/v3"
	//	"database/sql"
	//_ "github.com/mattn/go-sqlite3"
	//	_ "github.com/go-sql-driver/mysql"
)

var (
	//	db           *sql.DB = nil
	slideList    []Slide
	currentSlide = 0
	templates    = template.Must(template.ParseFiles("app/present.html"))
)

type Slide struct {
	Name  string
	Path  string
	Notes string
}

func (s Slide) String() string {
	return fmt.Sprintf("%s: %s {%s}", s.Name, s.Path, s.Notes)
}

func GetFileList(filename string) ([]Slide, error) {
	text, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	list := []Slide{}
	err = json.Unmarshal(text, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetSlide(name string) (*Slide, error) {
	fmt.Println(slideList)
	for i := range slideList {
		fmt.Println("name: ", slideList[i].Name)
		if name == slideList[i].Name {
			return &slideList[i], nil
		}
	}
	return nil, errors.New("no Slide by that name")
}

func SlideHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	slidenum := currentSlide
	if len(r.Form["index"]) != 0 {
		slidenum, err := strconv.Atoi(r.Form["index"][0])
		if err == nil {
			if 0 > slidenum || slidenum >= len(slideList) {
				err = errors.New("out of bounds")
			}
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	slide := slideList[slidenum]

	data, err := ioutil.ReadFile(slide.Path)
	if err != nil {
		w.Write([]byte("Could not find slide file"))
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	path := path.Clean(r.URL.Path)
	path = path[1:]
	text, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(404)
	} else {
		w.Write(text)
	}
}
func ControlHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	found := false
	for _, command := range r.Form["command"] {
		fmt.Println(command)
		switch {
		case command == "next":
			found = true
			currentSlide += 1
		case command == "last":
			found = true
			currentSlide = len(slideList) - 1
		case command == "previous":
			found = true
			currentSlide -= 1
		case command == "first":
			found = true
			currentSlide = 0
		}
	}
	if currentSlide < 0 {
		currentSlide = 0
	}
	if currentSlide >= len(slideList) {
		currentSlide = len(slideList) - 1
	}
	if found {
		w.Header().Set("Location", "/panel")
		w.WriteHeader(http.StatusFound)
		return
	}
	w.Write([]byte(fmt.Sprintf("%v", currentSlide)))
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("app/view.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(data)
}
func PanelHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("app/panel.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(data)
}
func PresentHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "present.html", slideList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func init() {
	/*	thatdb, err := sql.Open("mysql", "present:pass@/present")
		db = thatdb
		if err != nil {
			panic(err.Error())
		}
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS slideshows (id INT AUTO_INCREMENT, user VARCHAR (100), current_slide INT,  name VARCHAR (100),PRIMARY KEY(id))")
		if err != nil {
			panic(err.Error())
		}
		err = db.Ping()
		if err != nil {
			panic(err.Error())
		}*/
	//http.HandleFunc("/", Handler)

	list, err := GetFileList("slides.json")
	slideList = list
	//	log.Info(slideList)
	if err != nil {
		panic(err)
		//		log.Info("could not load slides: ", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/static/{path:.*}", StaticHandler).Methods("GET")
	r.HandleFunc("/slide", SlideHandler).Methods("GET")
	r.HandleFunc("/", ViewHandler).Methods("GET")
	r.HandleFunc("/present", PresentHandler).Methods("GET")
	r.HandleFunc("/control", ControlHandler).Methods("GET")
	r.HandleFunc("/panel", PanelHandler).Methods("GET")
	r.HandleFunc("/login", Handler).Methods("GET")
	r.HandleFunc("/in", CatchHandler).Methods("GET")
	//log.Info("Starting Present Server")
	http.Handle("/", r)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	//	c := appengine.NewContext(r)
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		panic(err)
	}
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	authUrl := config.AuthCodeURL("state-token", oauth2.AccessTypeOnline)

	w.Header().Set("Location", authUrl)
	w.WriteHeader(http.StatusFound)
	return
}
func CatchHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ctx := appengine.NewContext(r)
	//context.Background()
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		panic(err)
	}
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	code := r.FormValue("code")
	tok, err := config.Exchange(ctx, code)
	if err != nil {
		panic(err)
	}
	client := config.Client(ctx, tok)
	srv, err := drive.New(client)
	if err != nil {
		panic(err)
	}
	req, err := srv.Files.List().Do()
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "<ul>")
	files := []drive.File{}
	for _, f := range req.Files {
		if f.MimeType == "application/vnd.google-apps.presentation" {
			fmt.Fprintf(w, "<li>%v</li>\n", f)
			files = append(files, *f)
		}
	}
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "%v\n\n", files)
	pres, err := srv.Files.Export(files[0].Id, "image/svg").Download()
	if err != nil {
		panic(err)
	}
	contents, err := ioutil.ReadAll(pres.Body)
	fmt.Fprintf(w, "%v", contents)
	pres.Body.Close()
	/*	u := user.Current(c)
		if u == nil {
			url, err := user.LoginURL(c, r.URL.String())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Location", url)
			w.WriteHeader(http.StatusFound)
			return
		}*/
	/*	inst, err := db.Prepare("INSERT INTO slideshows(user, name) VALUES(?,?)")
		if err != nil {
			panic(err.Error())
		}
		inst.Exec(u.ID, "THING")*/
	fmt.Fprintf(w, "Hello")
	/*	rows, err := db.Query("SELECT * FROM slideshows where user = ?", u.ID)
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()
		var (
			id       int
			username string
			name     string
		)
		for rows.Next() {
			err := rows.Scan(&id, &username, &name)
			if err != nil {
				panic(err.Error())
			}
			fmt.Fprintf(w, "ROW %v %v", id, name)
		}*/
}
