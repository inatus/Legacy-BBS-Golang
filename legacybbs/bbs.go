package legacybbs

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"http"
	"template"
	"time"
	"strings"
)

type Entry struct {
	Name string
	Email string
	Title string
	Message string
	Date datastore.Time
	Ip string
}

type Entry_view struct {
	Name string
	Email string
	Title string
	Message string
	Date string
}

var sanitizing_char [][]string = [][]string{ { "&", "<", ">", "'", "\"" }, { "&amp;", "&lt;", "&gt;", "&#39;", "&quot;" } }

func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/write", write)
}

func handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	
	q := datastore.NewQuery("Entry").Order("-Date")
	dataCount, err := q.Count(c)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
	}
	
	entries := make([]Entry, 0, dataCount)
	if _, err := q.GetAll(c, &entries); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	
	entry_views := make([]Entry_view, dataCount)
	for i, entry := range entries {
		entry_views[i].Name = entry.Name
		entry_views[i].Email = entry.Email
		entry_views[i].Title = entry.Title
		entry_views[i].Message = strings.Replace(entry.Message, "\n", "<br />", -1)
		localTime := time.SecondsToLocalTime(int64(entry.Date) / 1000000)
		entry_views[i].Date = fmt.Sprintf("%04d/%02d/%02d %02d:%02d:%02d", localTime.Year, localTime.Month, localTime.Day, localTime.Hour, localTime.Minute, localTime.Second)		
	}
	
	var homeTemplate = template.Must(template.New("html").ParseFile("html/home.html"))
	if err := homeTemplate.Execute(w, entry_views); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
}

func write(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Not Found")
	}
	
	c := appengine.NewContext(r)
	
	var e Entry
	e.Name = r.FormValue("name")
	e.Email = r.FormValue("email")
	e.Title = r.FormValue("title")
	e.Message = r.FormValue("message")
	// Sanitizing
	for i := 0; i < len(sanitizing_char[0]); i++ {
		e.Name = strings.Replace(e.Name, sanitizing_char[0][i], sanitizing_char[1][i], -1)
		e.Email = strings.Replace(e.Email, sanitizing_char[0][i], sanitizing_char[1][i], -1)
		e.Title = strings.Replace(e.Title, sanitizing_char[0][i], sanitizing_char[1][i], -1)
		e.Message = strings.Replace(e.Message, sanitizing_char[0][i], sanitizing_char[1][i], -1)
	}
	e.Date = datastore.SecondsToTime(time.Seconds())
	e.Ip = r.RemoteAddr
	
	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Entry", nil), &e); err != nil {
		http.Error(w, "Internal Server Error: " + err.String(), http.StatusInternalServerError)
	}
	
	http.Redirect(w, r, "/", http.StatusFound)
}