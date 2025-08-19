package api

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/Derickcsantos/realtime-chat/internal/model"
    "github.com/Derickcsantos/realtime-chat/internal/repo"
)

func GetHotMessagesHandler(w http.ResponseWriter, r *http.Request) {
    limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
    if limit == 0 {
        limit = 20
    }
    msgs, err := repo.GetHotMessages(limit)
    if err != nil {
        http.Error(w, "Erro ao buscar mensagens quentes", 500)
        return
    }
    json.NewEncoder(w).Encode(msgs)
}

func GetColdMessagesHandler(w http.ResponseWriter, r *http.Request) {
    limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
    if limit == 0 {
        limit = 20
    }
    msgs, err := repo.GetColdMessages(limit)
    if err != nil {
        http.Error(w, "Erro ao buscar mensagens frias", 500)
        return
    }
    json.NewEncoder(w).Encode(msgs)
}

func PostMessageHandler(w http.ResponseWriter, r *http.Request) {
    var msg model.Message
    if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
        http.Error(w, "JSON inválido", 400)
        return
    }
    if err := repo.SaveHotMessage(msg); err != nil {
        http.Error(w, "Erro ao salvar mensagem quente", 500)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

func FlushHandler(w http.ResponseWriter, r *http.Request) {
    if err := repo.FlushHotToCold(); err != nil {
        http.Error(w, "Erro ao flush", 500)
        return
    }
    w.WriteHeader(http.StatusOK)
}