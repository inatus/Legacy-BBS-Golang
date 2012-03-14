package legacybbs

import (
	"appengine"
	"appengine/datastore"
	"appengine/taskqueue"
	"fmt"
	"http"
	"html"
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
    NameError bool
    MessageError bool
	Name string
	Email string
	Title string
	Message string
}

var sanitizing_char [][]string = [][]string{ { "&", "<", ">", "'", "\"" }, { "&amp;", "&lt;", "&gt;", "&#39;", "&quot;" } }

func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/task", task)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var emptyValidation Validation
		display(w, r, emptyValidation)
	} else if r.Method == "POST" {
		write(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Not Found")
	}
}

func display(w http.ResponseWriter, r *http.Request, v Validation) {
	c := appengine.NewContext(r)
	
	// Retrieves current page number from request parameter
    currentPage, _ := strconv.Atoi(r.FormValue("page"))
    if currentPage == 0 {
        currentPage = 1
    }
	
	// Retrieves entry count and get number of last page
	qCount := datastore.NewQuery("Entry")
	count, err := qCount.Count(c)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
	}
    lastPage :=  int(math.Ceil(float64(count) / 10.0))

	// Retrieves 10 entries which are diplayed in current page
	q := qCount.Order("-Date").Offset((currentPage - 1) * 10).Limit(10)
	dataCount, err := q.Count(c)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
	}
	entries := make([]Entry, 0, dataCount)
	if _, err := q.GetAll(c, &entries); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	
	// Makes view object for displayed html tamplate
    view := new(View)
    view.PreviousPage = currentPage - 1
    view.NextPage = currentPage + 1
    view.Pages = make([]Page, lastPage)
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
    if currentPage == lastPage {
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
	view.Errors = v
    
	// Sets view object to template and displays html
	var homeTemplate = template.Must(template.New("html").ParseFile("html/home.html"))
	if err := homeTemplate.Execute(w, view); err != nil {
		http.Error(w, "aa" + err.String(), http.StatusInternalServerError)
		return
	}
}

func write(w http.ResponseWriter, r *http.Request) {
	
	c := appengine.NewContext(r)
	
    // Retrieves form data
	var e Entry
	e.Name = r.FormValue("name")
	e.Email = r.FormValue("email")
	e.Title = r.FormValue("title")
	e.Message = r.FormValue("message")

    // Validates form data
    hasError := false
	var errors Validation
    if e.Name == "" {
        errors.NameError = true
        hasError = true
    }
    if e.Message == "" {
        errors.MessageError = true
        hasError = true
    }

	// Display errors in html if errors present
    if hasError {
		errors.Name = e.Name
		errors.Email = e.Email
		errors.Title = e.Title
		errors.Message = e.Message
        display(w, r, errors)
        return
    }

	// Sanitizes form strings
   	e.Name = html.EscapeString(e.Name)
   	e.Email = html.EscapeString(e.Email)
    e.Title = html.EscapeString(e.Title)
   	e.Message = html.EscapeString(e.Message)
   	e.Date = datastore.SecondsToTime(time.Seconds())
   	e.Ip = r.RemoteAddr
   	
	// Writes form data to datastore
   	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Entry", nil), &e); err != nil {
    	http.Error(w, "Internal Server Error: " + err.String(), http.StatusInternalServerError)
   	}
	
	// Add past notification email process to TaskQueue
	param := make(map[string][]string)
	param["name"] = []string{e.Name}
	param["title"] = []string{e.Title}
	param["message"] = []string{e.Message}
	task := taskqueue.NewPOSTTask("/task", param)
	if _, err := taskqueue.Add(c, task, ""); err != nil {
		http.Error(w, "Internal Server Error: " + err.String(), http.StatusInternalServerError)
		return
	}

    c.Infof(e.Name + " " + e.Ip)

	http.Redirect(w, r, "/", http.StatusFound)
}
