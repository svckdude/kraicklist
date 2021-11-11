package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	// "github.com/vcraescu/go-paginator"
	// "github.com/vcraescu/go-paginator/adapter"
)

func main() {
	// initialize searcher
	searcher := &Searcher{}
	err := searcher.Load("data.gz")
	if err != nil {
		log.Fatalf("unable to load search data due: %v", err)
	}
	// define http handlers
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/search", handleSearch(searcher))
	// define port, we need to set it as env for Heroku deployment
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	// start server
	fmt.Printf("Server is listening on %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatalf("unable to start server due: %v", err)
	}
}

func handleSearch(s *Searcher) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// fetch query string from query params
			q := r.URL.Query().Get("q")
			p, err := strconv.Atoi(r.URL.Query().Get("p"))
			// if len(q) == 0 {
			// 	w.WriteHeader(http.StatusBadRequest)
			// 	w.Write([]byte("missing search query in query params"))
			// 	return
			// }

			// search relevant records
			records, err := s.Search(q)

			// page records
			pageSize := 50
			start, end := paginate(p, pageSize, len(records))
			pagedSlice := records[start:end]

			totalPages := float64(len(records)) / float64(pageSize)
			var response PaginateRecord
			response.Records = pagedSlice
			response.Page = p
			response.TotalPages = int(math.Ceil(totalPages))

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			// output success response
			buf := new(bytes.Buffer)
			encoder := json.NewEncoder(buf)
			encoder.Encode(response)
			w.Header().Set("Content-Type", "application/json")
			w.Write(buf.Bytes())
		},
	)
}

type Searcher struct {
	records []Record
}

func (s *Searcher) Load(filepath string) error {
	// open file
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("unable to open source file due: %v", err)
	}
	defer file.Close()
	// read as gzip
	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("unable to initialize gzip reader due: %v", err)
	}
	// read the reader using scanner to contstruct records
	var records []Record
	cs := bufio.NewScanner(reader)
	for cs.Scan() {
		var r Record
		err = json.Unmarshal(cs.Bytes(), &r)
		if err != nil {
			continue
		}
		records = append(records, r)
	}
	s.records = records

	return nil
}

func (s *Searcher) Search(query string) ([]Record, error) {
	var result []Record
	query = strings.ToLower(query)

	for _, record := range s.records {
		title := strings.ToLower(record.Title)
		content := strings.ToLower(record.Content)

		if strings.Contains(title, query) || strings.Contains(content, query) {
			result = append(result, record)
		}
	}
	return result, nil
}

func paginate(pageNum int, pageSize int, sliceLength int) (int, int) {
	start := (pageNum - 1) * pageSize

	if start > sliceLength {
		start = sliceLength
	}

	end := start + pageSize
	if end > sliceLength {
		end = sliceLength
	}

	return start, end
}

type Record struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	ThumbURL  string   `json:"thumb_url"`
	Tags      []string `json:"tags"`
	UpdatedAt int64    `json:"updated_at"`
	ImageURLs []string `json:"image_urls"`
}

type PaginateRecord struct {
	Records    []Record
	Page       int
	TotalPages int
}
