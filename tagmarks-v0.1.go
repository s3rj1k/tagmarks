package main

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

//cat bookmarks.html | ./tagmarks.pl | sqlite3 tagmarks.db
//sqlite3
//.open tagmarks.db
//.quit

var dbPath string = "./tagmarks.db"
var sTags string = ""

func init() {
	tRow, err := sqliteQueryNthRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='tagmarks'", 1, 0)
	if err != nil {
		log.Panicln(err)
	} else {
		if tRow[0] != "CREATE VIRTUAL TABLE tagmarks USING fts4(\n\t\"url\" TEXT NOT NULL,\n\t\"name\" TEXT NOT NULL,\n\t\"date\" TEXT NOT NULL,\n\t\"tags\" TEXT DEFAULT ('NULL')\n)" {
			log.Panicln("sqlite db structure error!")
		}
	}
}

func main() {
	http.HandleFunc("/", GetHTML)
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

func sqliteExec(dbStr string) error {
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

func sqliteGetTags() ([]string, error) {
	slice := make([]string, 0)
	tRows, err := sqliteQuery("SELECT DISTINCT tags FROM tagmarks ORDER BY tags ASC", 1)
	if err != nil {
		return slice, err
	} else {
		var i int64
		var length int64 = int64(len(tRows))
		for i = 0; i < length; i++ {
			for j, _ := range tRows[i] {
				row := strings.Split(tRows[i][j], ",")
				for k, _ := range row {
					slice = append(slice, row[k])
				}
			}
		}
		slice = slice[:dedupe(sort.StringSlice(slice))]
		return slice, nil
	}
}

func sqliteQuery(dbStr string, tRowCellsAmount int) (map[int64]map[int]string, error) {
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

func sqliteQueryNthRow(dbStr string, tRowCellsAmount int, tRowOffset int64) ([]string, error) {
	tRow := make([]string, tRowCellsAmount)
	tRowPtrs := make([]interface{}, tRowCellsAmount)
	for i, _ := range tRow {
		tRowPtrs[i] = &tRow[i]
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return tRow, err
	}
	defer db.Close()
	if tRowOffset >= 0 {
		err = db.QueryRow(dbStr + " LIMIT 1 OFFSET " + strconv.FormatInt(tRowOffset, 10)).Scan(tRowPtrs...)
	} else {
		err = db.QueryRow(dbStr).Scan(tRowPtrs...)
	}
	switch {
	case err == sql.ErrNoRows:
		return tRow, err
	case err != nil:
		return tRow, err
	}
	return tRow, nil
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
	fmt.Println(sTags)
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
	tRows, err := sqliteQuery(dbStr, 4)
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
