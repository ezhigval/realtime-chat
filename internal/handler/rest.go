package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ezhigval/go-toolkit/httputil"
	"github.com/ezhigval/realtime-chat/internal/service"
	"github.com/go-chi/chi/v5"
)

type REST struct {
	svc *service.ChatService
}

func NewREST(svc *service.ChatService) *REST {
	return &REST{svc: svc}
}

func (h *REST) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, httputil.NewAppError(http.StatusBadRequest, "BAD_REQUEST", "invalid json", err))
		return
	}
	room, err := h.svc.CreateRoom(r.Context(), req.Name)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrEmptyName {
			status = http.StatusBadRequest
		}
		httputil.WriteError(w, httputil.NewAppError(status, "CREATE_FAILED", err.Error(), err))
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, room)
}

func (h *REST) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.svc.ListRooms(r.Context())
	if err != nil {
		httputil.WriteError(w, httputil.NewAppError(http.StatusInternalServerError, "INTERNAL", "list rooms failed", err))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, rooms)
}

func (h *REST) ListMessages(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, httputil.NewAppError(http.StatusBadRequest, "BAD_REQUEST", "invalid room id", err))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var before time.Time
	if b := r.URL.Query().Get("before"); b != "" {
		before, _ = time.Parse(time.RFC3339, b)
	}

	msgs, err := h.svc.ListMessages(r.Context(), roomID, limit, before)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrRoomNotFound {
			status = http.StatusNotFound
		}
		httputil.WriteError(w, httputil.NewAppError(status, "LIST_FAILED", err.Error(), err))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, msgs)
}

func (h *REST) ListPresence(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httputil.WriteError(w, httputil.NewAppError(http.StatusBadRequest, "BAD_REQUEST", "invalid room id", err))
		return
	}
	users, err := h.svc.ListPresence(r.Context(), roomID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrRoomNotFound {
			status = http.StatusNotFound
		}
		httputil.WriteError(w, httputil.NewAppError(status, "PRESENCE_FAILED", err.Error(), err))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"room_id": roomID, "users": users})
}
