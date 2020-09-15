package cmd

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/sensepost/gowitness/lib"
	"github.com/sensepost/gowitness/storage"
	"github.com/sensepost/gowitness/web"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	tmpl *template.Template
	rsDB *gorm.DB
)

// reportServeCmd represents the reportServe command
var reportServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts a web server to view screenshot reports",
	Run: func(cmd *cobra.Command, args []string) {
		log := options.Logger

		tmpl = template.Must(vfstemplate.ParseGlob(web.Assets, nil, "templates/*.html"))

		// db
		dbh, err := db.Get()
		if err != nil {
			log.Fatal().Err(err).Msg("could not gt db handle")
		}
		rsDB = dbh

		log.Info().Str("path", db.Path).Msg("db path")
		log.Info().Str("path", options.ScreenshotPath).Msg("screenshot path")

		// routes
		http.HandleFunc("/", indexHandler)
		http.HandleFunc("/table/", tableHandler)
		http.HandleFunc("/details", detailHandler)

		// static
		http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(web.Assets)))
		http.Handle("/screenshots/", http.StripPrefix("/screenshots", http.FileServer(http.Dir(options.ScreenshotPath))))

		log.Info().Str("address", options.ServerAddr).Msg("server listening")
		if err := http.ListenAndServe(options.ServerAddr, nil); err != nil {
			log.Fatal().Err(err).Msg("webserver failed")
		}
	},
}

func init() {
	reportCmd.AddCommand(reportServeCmd)

	reportServeCmd.Flags().StringVarP(&options.ServerAddr, "address", "a", "localhost:7171", "server listening address")
}

// detailHandler gets all of the details for a particular url id
func detailHandler(w http.ResponseWriter, r *http.Request) {

	d := strings.TrimSpace(r.URL.Query().Get("id"))
	if d == "" {
		http.Redirect(w, r, "/", 301)
		return
	}
	id, err := strconv.Atoi(d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var url storage.URL
	rsDB.
		Preload("Headers").
		Preload("TLS").
		Preload("TLS.TLSCertificates").
		Preload("TLS.TLSCertificates.DNSNames").
		First(&url, id)

	// fmt.Printf("%+v\n", url)

	t := tmpl.Lookup("detail.html")
	err = t.ExecuteTemplate(w, "detail", url)
	if err != nil {
		panic(err)
	}
}

// tableHandler handles the URL table view
func tableHandler(w http.ResponseWriter, r *http.Request) {

	var urls []storage.URL
	rsDB.Find(&urls)

	t := tmpl.Lookup("table.html")
	err := t.ExecuteTemplate(w, "table", urls)
	if err != nil {
		panic(err)
	}
}

// indexHandler handles the index page. this is the main gallery view
func indexHandler(w http.ResponseWriter, r *http.Request) {

	currPage, limit, err := getPageLimit(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pager := &lib.Pagination{
		DB:       rsDB,
		CurrPage: currPage,
		Limit:    limit,
	}

	var urls []storage.URL
	page, err := pager.Page(&urls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// fmt.Printf("%+v\n", currPage)

	t := tmpl.Lookup("gallery.html")
	err = t.ExecuteTemplate(w, "gallery", page)
	if err != nil {
		panic(err)
	}
}

// getPageLimit gets the limit and page query string values from a request
func getPageLimit(r *http.Request) (page int, limit int, err error) {

	pageS := strings.TrimSpace(r.URL.Query().Get("page"))
	limitS := strings.TrimSpace(r.URL.Query().Get("limit"))

	if pageS == "" {
		pageS = "-1"
	}
	if limitS == "" {
		limitS = "0"
	}

	page, err = strconv.Atoi(pageS)
	if err != nil {
		return
	}
	limit, err = strconv.Atoi(limitS)
	if err != nil {
		return
	}

	return
}
