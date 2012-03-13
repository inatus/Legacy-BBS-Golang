package legacybbs

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"http"
	"template"
	"time"
	"strings"
    "strconv"
    "math"
)

type Entry struct {
	Name string
	Email string
	Title string
	Message string
	Date datastore.Time
	Ip string
}

type View struct {
    Entries []Entry_view
    Pages []Page
    PreviousPage int
    NextPage int
    IsFirstPage bool
    IsLastPage bool
    IsError bool
    Errors Validation
}

type Entry_view struct {
	Name string
	Email string
	Title string
	Message string
	Date string
}

type Page struct {
    PageNum int
    IsCurrentPage bool
}

type Validation struct {
    Name bool
    Message bool
}

var sanitizing_char [][]string = [][]string{ { "&", "<", ">", "'", "\"" }, { "&amp;", "&lt;", "&gt;", "&#39;", "&quot;" } }

func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/write", write)
}

func handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

    currentPage, _ := strconv.Atoi(r.FormValue("page"))
    if currentPage == 0 {
        currentPage = 1
    }
	
	qCount := datastore.NewQuery("Entry")
	count, err := qCount.Count(c)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
	}

    maxPage :=  int(math.Ceil(float64(count) / 10.0))

	q := datastore.NewQuery("Entry").Order("-Date").Offset((currentPage - 1) * 10).Limit(10)
	dataCount, err := q.Count(c)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
	}
	
	entries := make([]Entry, 0, dataCount)
	if _, err := q.GetAll(c, &entries); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	
    view := new(View)
    view.PreviousPage = currentPage - 1
    view.NextPage = currentPage + 1
    view.Pages = make([]Page, maxPage)
    for i := 0; i < len(view.Pages); i++ {
        view.Pages[i].PageNum = i + 1
        if currentPage == i + 1 {
            view.Pages[i].IsCurrentPage = true
        } else {
            view.Pages[i].IsCurrentPage = false
        }
    }
    if currentPage == 1 {
        view.IsFirstPage = true
    }
    if currentPage == maxPage {
        view.IsLastPage = true
    }
	view.Entries = make([]Entry_view, dataCount)
	for i, entry := range entries {
		view.Entries[i].Name = entry.Name
		view.Entries[i].Email = entry.Email
		view.Entries[i].Title = entry.Title
		view.Entries[i].Message = strings.Replace(entry.Message, "\n", "<br />", -1)
		localTime := time.SecondsToLocalTime(int64(entry.Date) / 1000000)
		view.Entries[i].Date = fmt.Sprintf("%04d/%02d/%02d %02d:%02d:%02d", localTime.Year, localTime.Month, localTime.Day, localTime.Hour, localTime.Minute, localTime.Second)		
	}
	
	var homeTemplate = template.Must(template.New("html").ParseFile("html/home.html"))
	if err := homeTemplate.Execute(w, view); err != nil {
		http.Error(w, "aa" + err.String(), http.StatusInternalServerError)
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
