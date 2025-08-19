package model

type Message struct {
	ID   string `json:"id" bson:"_id,omitempty"`
    Type string `json:"type" bson:"type"`
    Text string `json:"text" bson:"text"`
    From string `json:"from" bson:"from"`
    Time int64  `json:"time" bson:"time"`
}