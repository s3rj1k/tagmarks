package main

//export GOROOT=$HOME/go && export GOPATH=$HOME/workspace-go && export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
//go build sql.go

import (
	_ "github.com/mxk/go-sqlite/sqlite3"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"bufio"
	"os"
	"regexp"
)

var sTags string = ""

func main() {
	http.HandleFunc("/main", MainWeb)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func dedupe(data sort.Interface) (n int) {
	if n = data.Len(); n < 2 {
		return n
	}
	sort.Sort(data)
	a, b := 0, 1
	for b < n {
		if data.Less(a, b) {
			a++
			if a != b {
				data.Swap(a, b)
			}
		}
		b++
	}
	return a + 1
}

func init() {
	f, err := os.Open("./bookmarks.html")
	if err != nil {
		log.Panicln(err)
	}
	db, err := sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	if err != nil {
		log.Panicln(err)
	}
	db.Exec("PRAGMA synchronous=OFF")
	db.Exec("DROP TABLE IF EXISTS tagmarks;")
	_, err = db.Exec("CREATE VIRTUAL TABLE tagmarks USING fts4(url, name, date, tags TEXT DEFAULT ('NULL'));")
	if err != nil {
		log.Panicln(err)
	}
	buf := bufio.NewScanner(f)
	reUrl := regexp.MustCompile("(<A HREF=\"[^\"]*\")")
	reName1 := regexp.MustCompile("(>[^\"]*</A>)")
	reName2 := regexp.MustCompile(">([^>^<]*)<")
	reDate := regexp.MustCompile("(LAST_MODIFIED=\"[^\"]*\")")
	reTags := regexp.MustCompile("(TAGS=\"[^\"]*\")")
	reQuote := regexp.MustCompile("\"([^\"]*)\"")
	for buf.Scan() {
		url := reUrl.FindString(buf.Text())
		url = reQuote.FindString(url)
		url = strings.Trim(url, "\"")
		if len(url) > 0 {
			name := reName1.FindString(buf.Text())
			name = reName2.FindString(name)
			name = strings.TrimSuffix(name, "<")
			name = strings.TrimPrefix(name, ">")
			if len(name) <= 0 {name = "NULL"}
			date := reDate.FindString(buf.Text())
			date = reQuote.FindString(date)
			date = strings.Trim(date, "\"")
			tags := reTags.FindString(buf.Text())
			tags = reQuote.FindString(tags)
			tags = strings.Trim(tags, "\"")
			if len(tags) <= 0 {tags = "NULL"}
			_, err = db.Exec("INSERT INTO tagmarks(url, name, date, tags) VALUES(\"" + url + "\",\"" + name + "\",\"" + date + "\",\"" + tags + "\");")
			if err != nil {
				log.Println(err)
			}
		}
	}
	if err := buf.Err(); err != nil {
		log.Fatal(err)
	}
}

func MainWeb(w http.ResponseWriter, r *http.Request) {
	//curl -X POST -d tag="электроника" -d tag="diy" http://localhost:8080/
	//http://requestb.in
	if r.Method == "POST" {
		r.ParseForm()
		if aTags := r.PostForm["tag"]; len(aTags) > 0 {
			sTags = strings.Join(aTags, "\" \"")
			sTags = "\"" + sTags + "\""
		} else {
			sTags = ""
		}
	}
	var dbStr string
	if len(strings.Trim(sTags, "\"")) > 0 {
		dbStr = "SELECT url, name, date, tags FROM tagmarks WHERE tags MATCH '" + sTags + "' GROUP BY url ORDER BY date DESC, name ASC"
	} else {
		dbStr = "SELECT url, name, date, tags FROM tagmarks ORDER BY date DESC, name ASC"
	}
	fmt.Fprintln(w, "<!DOCTYPE html>")
	fmt.Fprintln(w, "<html>")
	fmt.Fprintln(w, "<head>")
	fmt.Fprintln(w, "<meta charset=\"utf-8\" />")
	fmt.Fprintln(w, "<meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge,chrome=1\" />")
	fmt.Fprintln(w, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0, maximum-scale=1.0\">")
	fmt.Fprintln(w, "<title>Tagmarks</title>")
	fmt.Fprintln(w, "<link rel=\"stylesheet\" type=\"text/css\" class=\"ui\" href=\"http://cdnjs.cloudflare.com/ajax/libs/semantic-ui/0.16.1/css/semantic.min.css\">")
	fmt.Fprintln(w, "<link href=\"http://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.1.0/css/font-awesome.min.css\" rel=\"stylesheet\">")
	fmt.Fprintln(w, "<script src=\"http://cdnjs.cloudflare.com/ajax/libs/jquery/2.1.1/jquery.min.js\"></script>")
	fmt.Fprintln(w, "<script src=\"http://cdnjs.cloudflare.com/ajax/libs/semantic-ui/0.16.1/javascript/semantic.min.js\"></script>")
	fmt.Fprintln(w, "<style>")
	fmt.Fprintln(w, ".url { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 80%; }")
	fmt.Fprintln(w, "</style>")
	fmt.Fprintln(w, "</head>")
	db, err := sql.Open("sqlite3", "file:memdb1?mode=memory&cache=shared")
	if err != nil {
		log.Panicln(err)
	}
	fmt.Fprintln(w, "<body>")
	fmt.Fprintln(w, "<div class=\"ui basic segment relaxed list\">")
	rows, err := db.Query(dbStr)
	if err != nil {
		log.Panicln(err)
	}
	var url, name, date, tags string
	allTags := make([]string, 0)
	for rows.Next() {
		if err := rows.Scan(&url, &name, &date, &tags); err != nil {
			log.Panicln(err)
		}
		t, _ := strconv.ParseInt(date, 10, 64)
		if date != time.Unix(t, 0).Format("2006-01-02") {
			date = time.Unix(t, 0).Format("2006-01-02")
			fmt.Fprintln(w, "<div class=\"ui ribbon label\"><i class=\"calendar icon\">&nbsp;</i>"+date+":</div>")
		}
		fmt.Fprintln(w, "<div class=\"item\">")
		fmt.Fprintln(w, "<div class=\"content url\">")
		fmt.Fprintln(w, "<i class=\"bookmark icon\"></i><a class=\"header url popup\" href=\""+url+"\" data-html=\"<div class='ui horizontal list'><a class='item'><i class='small tag icon'></i>"+tags+"</a></div>\" data-variation=\"small\">"+name+"</a>")
		fmt.Fprintln(w, "</div>")
		fmt.Fprintln(w, "</div>")
		arr := strings.Split(tags, ",")
		for k, _ := range arr {
			allTags = append(allTags, arr[k])
		}
	}
	if err := rows.Err(); err != nil {
		log.Panicln(err)
	}
	fmt.Fprintln(w, "</div>")
	fmt.Fprintln(w, "<div class=\"ui right sidebar vertical menu\">")
	fmt.Fprintln(w, "<div class=\"header item\"><i class=\"tags icon\"></i>Tags</div>")
	fmt.Fprintln(w, "<form id=\"tags_form\" action=\"/main\" method=\"POST\" class=\"ui form item tags_form\">")
	allTags = allTags[:dedupe(sort.StringSlice(allTags))]
	for i := 0; i < len(allTags); i++ {
		if len(strings.Trim(sTags, "\"")) > 0 {
			if strings.Contains(sTags, allTags[i]) {
				fmt.Fprintln(w, "<div class=\"inline field\">")
				fmt.Fprintln(w, "<div class=\"ui slider checkbox\">")
				fmt.Fprintln(w, "<input checked=\"checked\" id=\"tag_"+allTags[i]+"\" class=\"tag\" name=\"tag\" value=\""+allTags[i]+"\" type=\"checkbox\">")
				fmt.Fprintln(w, "<label for=\"tag_"+allTags[i]+"\">"+allTags[i]+"</label>")
				fmt.Fprintln(w, "<script>")
				fmt.Fprintln(w, "$(\"#tag_"+allTags[i]+"\").checkbox('enable');")
				fmt.Fprintln(w, "</script>")
				fmt.Fprintln(w, "</div>")
				fmt.Fprintln(w, "</div>")
			} else {
				fmt.Fprintln(w, "<div class=\"inline field\">")
				fmt.Fprintln(w, "<div class=\"ui slider checkbox\">")
				fmt.Fprintln(w, "<input id=\"tag_"+allTags[i]+"\" class=\"tag\" name=\"tag\" value=\""+allTags[i]+"\" type=\"checkbox\">")
				fmt.Fprintln(w, "<label for=\"tag_"+allTags[i]+"\">"+allTags[i]+"</label>")
				fmt.Fprintln(w, "</div>")
				fmt.Fprintln(w, "</div>")
			}
		} else {
			fmt.Fprintln(w, "<div class=\"inline field\">")
			fmt.Fprintln(w, "<div class=\"ui slider checkbox\">")
			fmt.Fprintln(w, "<input id=\"tag_"+allTags[i]+"\" class=\"tag\" name=\"tag\" value=\""+allTags[i]+"\" type=\"checkbox\">")
			fmt.Fprintln(w, "<label for=\"tag_"+allTags[i]+"\">"+allTags[i]+"</label>")
			fmt.Fprintln(w, "</div>")
			fmt.Fprintln(w, "</div>")
		}
	}
	fmt.Fprintln(w, "<!--<input type=\"submit\" value=\"Submit\" class=\"ui blue button submit\"/>-->")
	fmt.Fprintln(w, "</form>")
	fmt.Fprintln(w, "</div>")
	fmt.Fprintln(w, "<script>")
	fmt.Fprintln(w, "$('.sidebar')")
	fmt.Fprintln(w, ".sidebar('show')")
	fmt.Fprintln(w, ";")
	fmt.Fprintln(w, "$('.tag')")
	fmt.Fprintln(w, ".on('change', function() {")
	fmt.Fprintln(w, "//$('.submit').click()")
	fmt.Fprintln(w, "$('.tags_form').submit()")
	fmt.Fprintln(w, ";")
	fmt.Fprintln(w, "})")
	fmt.Fprintln(w, ";")
	fmt.Fprintln(w, "$('.popup')")
	fmt.Fprintln(w, ".popup({")
	fmt.Fprintln(w, "on: 'hover'")
	fmt.Fprintln(w, "})")
	fmt.Fprintln(w, ";")
	fmt.Fprintln(w, "</script>")
	fmt.Fprintln(w, "</body>")
	fmt.Fprintln(w, "</html>")
}
