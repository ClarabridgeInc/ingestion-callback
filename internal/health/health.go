package health

import (
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	*zap.Logger
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Write([]byte("OK"))
	default:
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	}
}
