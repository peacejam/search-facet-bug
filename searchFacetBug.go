package searchFacetBug

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"google.golang.org/appengine"
	"google.golang.org/appengine/search"
)

func init() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addHandler)
}

//Doc is a struct
type Doc struct {
	Title string
	F     float64 `search:",facet"`
}

//Save takes the fields from a Doc and saves them into the search index
func (d *Doc) Save() ([]search.Field, *search.DocumentMetadata, error) {
	fields := []search.Field{{Name: "Title", Value: d.Title}}
	md := &search.DocumentMetadata{
		Facets: []search.Facet{{Name: "F", Value: d.F}},
	}
	return fields, md, nil
}

//Load takes the slice of fields corresponding to a project and loads them into a SearchDocument
func (d *Doc) Load(fieldSlice []search.Field, md *search.DocumentMetadata) error {
	for _, facet := range md.Facets {
		if facet.Name != "F" {
			return fmt.Errorf("unknown facet %q", facet.Name)
		}
		fFloat, ok := facet.Value.(float64)
		if !ok {
			return fmt.Errorf("Converting from interface to float64")
		}
		d.F = fFloat
	}

	for _, field := range fieldSlice {
		if field.Name != "Title" {
			return fmt.Errorf("unknown title %q", field.Name)
		}
		titleAtom, ok := field.Value.(string)
		if !ok {
			return fmt.Errorf("Converting from interface to string")
		}
		d.Title = titleAtom
	}

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	docs := []*Doc{}
	searchIndex, err := search.Open("global")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t := searchIndex.Search(c, "", &search.SearchOptions{
		Facets: []search.FacetSearchOption{},
	})
	for {
		var doc Doc
		_, err := t.Next(&doc)
		if err == search.Done {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		docs = append(docs, &doc)
	}

	var tpl = template.Must(template.New("name").Parse(homeTpl))
	err = tpl.Execute(w, docs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	f, err := strconv.ParseFloat(r.FormValue("f"), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc := Doc{
		Title: r.FormValue("title"),
		F:     f,
	}

	c := appengine.NewContext(r)
	searchIndex, err := search.Open("global")
	_, err = searchIndex.Put(c, "", &doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

const homeTpl = `
<!DOCTYPE html>
<html>
<head></head>
<body>

	<h3>Documents in the search index</h3>
	<ul>
		{{ range . }}
			<li>
				<div>Title: {{ .Title }}</div>
				<div>F: {{ .F }}</div>
			</li>
		{{ end }}
	</ul>

	<form action="/add" method="POST">
		<h4>Add a document to the search index</h4>
		<input type="text" name="title" placeholder="Title" />
		<input type="number" name="f" placeholder="Facet (number)" />
		<input type="submit" value="Add" />
	</form>

</body>
</html>
`
