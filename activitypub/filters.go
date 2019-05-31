package activitypub

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	"github.com/mariusor/qstring"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Hash
type Hash string

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

const (
	ActorsType     = h.CollectionType("actors")
	ActivitiesType = h.CollectionType("activities")
	ObjectsType    = h.CollectionType("objects")
)

var validActivityCollection = []h.CollectionType{
	ActorsType,
	ActivitiesType,
	ObjectsType,
}

func getValidActivityCollection(typ string) h.CollectionType {
	for _, t := range validActivityCollection {
		if strings.ToLower(typ) == string(t) {
			return t
		}
	}
	return h.Unknown
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid one for handling Activities
func ValidActivityCollection(typ string) bool {
	return getValidActivityCollection(typ) != h.Unknown || h.ValidActivityCollection(typ) || h.ValidObjectCollection(typ)
}

// Filters
type Filters struct {
	Actor        as.Actor                    `qstring:"-"`
	Owner        as.Actor                    `qstring:"-"`
	Collection   h.CollectionType            `qstring:"-"`
	Key          []Hash                      `qstring:"id,omitempty"`
	ItemKey      []Hash                      `qstring:"objectid,omitempty"`
	Type         []as.ActivityVocabularyType `qstring:"type"`
	AttributedTo []Hash                      `qstring:"attributedTo,omitempty"`
	FollowedBy   []Hash                      `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan    time.Time                   `qstring:"olderThan,omitempty"`
	NewerThan    time.Time                   `qstring:"newerThan,omitempty"`
	Page         int                         `qstring:"page,omitempty"`
	MaxItems     int                         `qstring:"maxItems,omitempty"`
}

// IRIs returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Types() []as.ActivityVocabularyType {
	return f.Type
}

// IRIs returns a list of IRIs to filter against
func (f Filters) IRIs() []as.IRI {
	ret := make([]as.IRI, len(f.ItemKey))
	for i, k := range f.ItemKey {
		ret[i] = as.IRI(k)
	}
	return ret
}

// TODO(marius): move this somewhere else. Or replace it with something that makes more sense.
var Secure = false

func copyActivityFilters(dst *Filters, src Filters) {
	dst.Type = src.Type
	dst.Key = src.Key
	dst.ItemKey = src.ItemKey
	dst.Type = src.Type
	dst.AttributedTo = src.AttributedTo
	dst.FollowedBy = src.FollowedBy
	dst.OlderThan = src.OlderThan
	dst.NewerThan = src.NewerThan
	dst.Page = src.Page
	dst.MaxItems = src.MaxItems
}

func query(f *Filters) string {
	res := ""

	var u url.Values
	var err error
	if u, err = qstring.Marshal(f); err != nil {
		return ""
	}

	if len(u) > 0 {
		res = "?" + u.Encode()
	}
	return res
}

// QueryString
func (f *Filters) QueryString() string {
	return query(f)
}

// BasePage
func (f *Filters) BasePage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *f)
	return b
}

// CurrentPage
func (f *Filters) CurrentPage() Paginator {
	return f
}

// NextPage
func (f *Filters) NextPage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *f)
	b.Page += 1
	return b
}

// PrevPage
func (f *Filters) PrevPage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *f)
	b.Page -= 1
	return b
}

// FirstPage
func (f *Filters) FirstPage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *f)
	b.Page = 1
	return b
}

// CurrentIndex
func (f *Filters) CurrentIndex() int {
	return f.Page
}

type Filterable interface {
	GetWhereClauses() ([]string, []interface{})
	GetLimit() string
}

func (f *Filters) GetWhereClauses() ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	var counter = 1

	keys := f.Key
	if len(keys) > 0 {
		keyWhere := make([]string, 0)
		for _, hash := range keys {
			keyWhere = append(keyWhere, fmt.Sprintf(`"key" ~* $%d`, counter))
			values = append(values, interface{}(hash))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}
	types := f.Types()
	if len(types) > 0 {
		keyWhere := make([]string, 0)
		for _, typ := range types {
			keyWhere = append(keyWhere, fmt.Sprintf(`"type" = $%d`, counter))
			values = append(values, interface{}(typ))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}

	validAudienceType := false
	for _, typ := range f.Type {
		validAudienceType = as.ActivityTypes.Contains(typ) || as.ObjectTypes.Contains(typ) || as.ActorTypes.Contains(typ)
	}

	iris := f.IRIs()
	if len(iris) > 0 {
		keyWhere := make([]string, 0)
		for _, key := range iris {
			if _, err := url.ParseRequestURI(key.String()); err != nil {
				// not a valid iri
				keyWhere = append(keyWhere, fmt.Sprintf(`"key" ~* $%d`, counter))
			} else {
				if len(f.Type) == 1 && f.Type[0] == as.LinkType {
					keyWhere = append(keyWhere, fmt.Sprintf(`"raw"::text ~* $%d`, counter))
				} else if validAudienceType {
					// For Link type we need to search the full raw column
					keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'id' = $%d`, counter))
				}
				keyWhere = append(keyWhere, fmt.Sprintf(`"iri" = $%d`, counter))
			}
			values = append(values, interface{}(key))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}

	// TODO(marius): this looks cumbersome - we need to abstract the audience to something easier to query.
	if validAudienceType {
		keyWhere := make([]string, 0)
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'to' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'cc' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'bto' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'bcc' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'audience' ~* $%d`, counter))
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
		if f.Actor == nil {
			values = append(values, interface{}(Public))
		} else {
			values = append(values, interface{}(f.Actor.GetID()))
		}
	}

	if len(clauses) == 0 {
		clauses = append(clauses, " true")
	}

	return clauses, values
}

func (f Filters) GetLimit() string {
	if f.MaxItems == 0 {
		return ""
	}
	limit := fmt.Sprintf(" LIMIT %d", f.MaxItems)
	if f.Page >= 1 {
		limit = fmt.Sprintf("%s OFFSET %d", limit, f.MaxItems*(f.Page-1))
	}
	return limit
}

const MaxItems = 100
var ErrNotFound = func(s string) error {
	return errors.Newf(fmt.Sprintf("%s not found", s))
}

// FromRequest loads the filters we use for generating storage queries from the HTTP request
func FromRequest(r *http.Request) (Filterable, error) {
	f := &Filters{}
	if err := qstring.Unmarshal(r.URL.Query(), f); err != nil {
		return f, err
	}
	f.MaxItems = MaxItems

	f.Collection = h.Typer.Type(r)

	if f.MaxItems == 0 {
		f.MaxItems = MaxItems
	}
	if f.Page == 0 {
		f.Page = 1
	}

	return f, nil
}