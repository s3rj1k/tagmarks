package main

//export GOROOT=$HOME/go
//export GOPATH=$HOME/workspace-go
//export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
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
)

var sTags string = ""

func main() {
	http.HandleFunc("/", GetHTML)
	http.HandleFunc("/createDB/", CreateDB)
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

func sqliteExec(dbPath string, dbStr string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(dbStr)
	if err != nil {
		return err
	}
	return nil
}

func sqliteQuery(dbPath string, dbStr string, tRowCellsAmount int) (map[int64]map[int]string, error) {
	tRow := make([]string, tRowCellsAmount)
	tRowPtrs := make([]interface{}, tRowCellsAmount)
	tRows := make(map[int64]map[int]string)
	var rowN int64 = 0
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return tRows, err
	}
	defer db.Close()
	rows, err := db.Query(dbStr)
	if err != nil {
		return tRows, err
	}
	for rows.Next() {
		tRows[rowN] = make(map[int]string)
		for i, _ := range tRow {
			tRowPtrs[i] = &tRow[i]
		}
		if err := rows.Scan(tRowPtrs...); err != nil {
			return tRows, err
		}
		for i, _ := range tRow {
			tRows[rowN][i] = tRow[i]
		}
		rowN++
	}
	if err := rows.Err(); err != nil {
		return tRows, err
	}
	return tRows, nil
}

func CreateDB(w http.ResponseWriter, r *http.Request) {
	//curl -X POST -d mozDB="/home/user/.mozilla/firefox/o8o3gyyl.default/places.sqlite" http://localhost:8080/createDB/
	if r.Method == "POST" {
		mozDbPath := r.FormValue("mozDB")
		log.Println(mozDbPath)
		sqliteExec("./tagmarks.db", "DROP TABLE tagmarks;")
		err := sqliteExec("./tagmarks.db", "CREATE VIRTUAL TABLE tagmarks USING fts4(url, name, date, tags TEXT DEFAULT ('NULL'));")
		if err != nil {
			log.Panicln(err)
		} else {
			mozUrlIDs, err := sqliteQuery(mozDbPath, "SELECT fk FROM moz_bookmarks WHERE type=1 ORDER BY lastModified;", 1)
			if err != nil && len(mozUrlIDs) != 0 && len(mozUrlIDs[0]) != 1 {
				log.Panicln(err)
			} else {
				var i int64
				var mozUrlIDsLength int64 = int64(len(mozUrlIDs))
				var mozTagsList string
				for i = 0; i < mozUrlIDsLength; i++ {
					mozUrlID := mozUrlIDs[i][0]
					mozUrl, err := sqliteQuery(mozDbPath, "SELECT url FROM moz_places WHERE id=" + mozUrlID + " LIMIT 1;", 1)
					if err != nil && len(mozUrl) != 0 && len(mozUrl[0]) != 1 {
						log.Panicln(err)
					}
					mozTitle, err := sqliteQuery(mozDbPath, "SELECT title FROM moz_bookmarks WHERE fk=" + mozUrlID + " AND title!='';", 1)
					if err != nil && len(mozTitle) != 0 && len(mozTitle[0]) != 1 {
						log.Panicln(err)
					}
					mozDate, err := sqliteQuery(mozDbPath, "SELECT lastModified FROM moz_bookmarks WHERE fk=" + mozUrlID + " ORDER BY lastModified DESC;", 1)
					if err != nil && len(mozDate) != 0 && len(mozDate[0]) != 1 {
						log.Panicln(err)
					}
					mozTagIDs, err := sqliteQuery(mozDbPath, "SELECT parent FROM moz_bookmarks WHERE fk=" + mozUrlID + " AND title IS NULL;", 1)
					if err != nil && len(mozTagIDs) != 0 && len(mozTagIDs[0]) != 1 {
						log.Panicln(err)
					} else {
						mozTagIDsSlice := make([]string, 0)
						for  _, mozTagID := range mozTagIDs {
							mozTagIDsSlice = append(mozTagIDsSlice, mozTagID[0])
						}
						mozTagIDsList := strings.Join(mozTagIDsSlice, ",")
						mozTags, err := sqliteQuery(mozDbPath, "SELECT title FROM moz_bookmarks WHERE id IN (" + mozTagIDsList + ");", 1)
						if err != nil && len(mozTags) != 0 && len(mozTags[0]) != 1 {
							log.Panicln(err)
						} else {
							mozTagsSlice := make([]string, 0)
							for  _, mozTag := range mozTags {
								mozTagsSlice = append(mozTagsSlice, mozTag[0])
							}
							mozTagsSlice = mozTagsSlice[:dedupe(sort.StringSlice(mozTagsSlice))]
							mozTagsList = strings.Join(mozTagsSlice, ",")
						}
					}
					if len(mozTagsList) <= 0 {mozTagsList = "NULL"}
					mozTitle[0][0] = strings.Replace(mozTitle[0][0], "\"", "″", -1)
					//log.Println(mozUrl[0][0] + ";" + mozTitle[0][0] + ";" + mozDate[0][0]  + ";" + mozTagsList)
					err = sqliteExec("./tagmarks.db", "INSERT INTO tagmarks(url, name, date, tags) VALUES(\"" + mozUrl[0][0] + "\",\"" + mozTitle[0][0] + "\",\"" + mozDate[0][0] + "\",\"" + mozTagsList + "\");")
					if err != nil {
						log.Println(err)
						log.Println(mozUrl[0][0] + ";" + mozTitle[0][0] + ";" + mozDate[0][0]  + ";" + mozTagsList)
					}
				}
			}
			log.Println("tagmarks.db created")
		}
	}
}

func GetHTML(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintln(w, "<!doctype html>")
	fmt.Fprintln(w, "<html>")
	fmt.Fprintln(w, "<head>")
	fmt.Fprintln(w, "<meta charset='utf-8'>")
	fmt.Fprintln(w, "<meta name='viewport' content='width=device-width, initial-scale=1'>")
	fmt.Fprintln(w, "<title>TagMarks</title>")
	fmt.Fprintln(w, "<link rel='stylesheet' href='http://yui.yahooapis.com/pure/0.5.0/pure-min.css'>")
	fmt.Fprintln(w, "<style>")
	fmt.Fprintln(w, ".header { border-bottom: 1px solid #dedede; margin: 10px 0 3px 0; padding: 0; }")
	fmt.Fprintln(w, ".tags { color: #dedede; display: block; font-size: 90%; }")
	fmt.Fprintln(w, ".url { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 80%; }")
	fmt.Fprintln(w, "</style>")
	fmt.Fprintln(w, "</head>")
	tRows, err := sqliteQuery("./tagmarks.db", dbStr, 4)
	if err != nil && len(tRows) != 0 && len(tRows[0]) != 4 {
		log.Panicln(err)
	} else {
		var i int64
		var tRowsLength int64 = int64(len(tRows))
		var allTagsLength int64
		var date string
		allTags := make([]string, 0)
		for i = 0; i < tRowsLength; i++ {
			arr := strings.Split(tRows[i][3], ",")
			for k, _ := range arr {
				allTags = append(allTags, arr[k])
			}
		}
		allTags = allTags[:dedupe(sort.StringSlice(allTags))]
		allTagsLength = int64(len(allTags))
		fmt.Fprintln(w, "<body>")
		fmt.Fprintln(w, "<div>")
		fmt.Fprintln(w, "<h3 class='header'>Tags:</h3>")
		fmt.Fprintln(w, "<form id='tags' name='tags' action='/' method='POST' class='pure-form'>")
		fmt.Fprintln(w, "<fieldset>")
		for i = 0; i < allTagsLength; i++ {
			if len(strings.Trim(sTags, "\"")) > 0 {
				if strings.Contains(sTags, allTags[i]) {
					fmt.Fprintln(w, "<label for='tag_"+allTags[i]+"'>")
					fmt.Fprintln(w, "<input id='tag_"+allTags[i]+"' type='checkbox' name='tag' value='"+allTags[i]+"' checked>"+allTags[i])
					fmt.Fprintln(w, "</label>")
				} else {
					fmt.Fprintln(w, "<label for='tag_"+allTags[i]+"'>")
					fmt.Fprintln(w, "<input id='tag_"+allTags[i]+"' type='checkbox' name='tag' value='"+allTags[i]+"'>"+allTags[i])
					fmt.Fprintln(w, "</label>")
				}
			} else {
				fmt.Fprintln(w, "<label for='tag_"+allTags[i]+"'>")
				fmt.Fprintln(w, "<input id='tag_"+allTags[i]+"' type='checkbox' name='tag' value='"+allTags[i]+"'>"+allTags[i])
				fmt.Fprintln(w, "</label>")
			}
		}
		fmt.Fprintln(w, "</fieldset>")
		fmt.Fprintln(w, "<input type='submit' form='tags' value='Отправить форму' class='pure-button pure-button-primary'>")
		fmt.Fprintln(w, "</form>")
		for i = 0; i < tRowsLength; i++ {
			t, _ := strconv.ParseInt(tRows[i][2], 10, 64)
			if date != time.Unix(t, 0).Format("2006-01-02") {
				date = time.Unix(t, 0).Format("2006-01-02")
				fmt.Fprintln(w, "<h3 class='header'>"+date+":</h3>")
			}
			fmt.Fprintln(w, "<div class='url'>")
			fmt.Fprintln(w, "<a href='"+tRows[i][0]+"'>"+tRows[i][1]+"</a>")
			fmt.Fprintln(w, "<i class='tags'> "+tRows[i][3]+"</i>")
			fmt.Fprintln(w, "</div>")
		}
		fmt.Fprintln(w, "</div>")
		fmt.Fprintln(w, "</body>")
	}
	fmt.Fprintln(w, "</html>")
}
