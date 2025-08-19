package repo

import (
    "context"
    "encoding/json"
    "log"
    "time"

    "github.com/Derickcsantos/realtime-chat/internal/model"
    "github.com/Derickcsantos/realtime-chat/internal/store"
    "go.mongodb.org/mongo-driver/bson"
)

const redisKey = "hot_messages"

func SaveHotMessage(msg model.Message) error {
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    return store.RedisClient.LPush(store.Ctx, redisKey, data).Err()
}

func GetHotMessages(limit int64) ([]model.Message, error) {
    vals, err := store.RedisClient.LRange(store.Ctx, redisKey, 0, limit-1).Result()
    if err != nil {
        return nil, err
    }
    var msgs []model.Message
    for _, v := range vals {
        var m model.Message
        _ = json.Unmarshal([]byte(v), &m)
        msgs = append(msgs, m)
    }
    return msgs, nil
}

func SaveColdMessage(msg model.Message) error {
    _, err := store.MongoMsgs.InsertOne(context.Background(), msg)
    return err
}

func GetColdMessages(limit int64) ([]model.Message, error) {
    cur, err := store.MongoMsgs.Find(context.Background(), bson.D{}, nil)
    if err != nil {
        return nil, err
    }
    defer cur.Close(context.Background())
    var msgs []model.Message
    for cur.Next(context.Background()) {
        var m model.Message
        _ = cur.Decode(&m)
        msgs = append(msgs, m)
        if int64(len(msgs)) >= limit {
            break
        }
    }
    return msgs, nil
}

// Move hot messages to cold (flush)
func FlushHotToCold() error {
    msgs, err := GetHotMessages(100)
    if err != nil {
        return err
    }
    for _, m := range msgs {
        _ = SaveColdMessage(m)
    }
    return store.RedisClient.Del(store.Ctx, redisKey).Err()
}

func init() {
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()
        for range ticker.C {
            if err := FlushHotToCold(); err != nil {
                log.Println("Erro ao flush do Redis para Mongo:", err)
            } else {
                log.Println("Flush do Redis para Mongo realizado com sucesso!")
            }
        }
    }()
}