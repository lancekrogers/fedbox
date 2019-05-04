package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"net/http"
)

// HandleServerRequest handles server to server (S2S) POST requests to an ActivityPub Actor's inbox
func HandleServerRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	return as.IRI(""), http.StatusNotImplemented, errors.NotImplementedf("not implemented")
}
