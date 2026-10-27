package label_model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Label struct {
	Id           string `json:"id" gorm:"type:uuid;primaryKey"`
	InstanceID   string `json:"instance_id" gorm:"type:uuid"`
	LabelID      string `json:"label_id"`
	LabelName    string `json:"label_name"`
	LabelColor   string `json:"label_color"`
	PredefinedId string `json:"predefined_id"`
}

func (m *Label) BeforeCreate(tx *gorm.DB) (err error) {
	m.Id = uuid.New().String()
	return
}

// LabelChat is the association between a WhatsApp native label and a chat JID.
// It mirrors the app state label_jid mutations so contacts can be listed by label
// without asking WhatsApp for the whole contact list again.
type LabelChat struct {
	Id         string    `json:"id" gorm:"type:uuid;primaryKey"`
	InstanceID string    `json:"instance_id" gorm:"type:uuid;uniqueIndex:idx_label_chats_instance_label_jid"`
	LabelID    string    `json:"label_id" gorm:"uniqueIndex:idx_label_chats_instance_label_jid"`
	JID        string    `json:"jid" gorm:"column:jid;type:text;uniqueIndex:idx_label_chats_instance_label_jid"`
	Labeled    bool      `json:"labeled"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (m *LabelChat) BeforeCreate(tx *gorm.DB) (err error) {
	if m.Id == "" {
		m.Id = uuid.New().String()
	}
	return
}
