package entity

import (
	"fmt"

	"github.com/Hello-Storage/hello-storage-proxy/internal/db"
	"gorm.io/gorm"
)

type UserDetail struct {
	ID              uint         `gorm:"primarykey" json:"id"`
	StorageUsed     uint         `                  json:"storage_used"` // bytes format
	Subscription    Subscription `                  json:"subscription"`
	ReferredBy      uint         `gorm:"foreignKey:UserID;references:ID" json:"referred_by"`
	ReferralStorage uint         `json:"referral_storage"` // bytes format
	UserID          uint
}

// TableName returns the entity table name.
func (UserDetail) TableName() string {
	return "user_details"
}

func (m *UserDetail) Create() error {
	return db.Db().Create(m).Error
}

func (m *UserDetail) TxCreate(tx *gorm.DB) error {
	return tx.Create(m).Error
}

func (m *UserDetail) Save() error {
	return db.Db().Save(m).Error
}

// Update a face property in the database.
func (m *UserDetail) Update(attr string, value interface{}) error {
	if m.ID == 0 {
		return fmt.Errorf("empty id")
	}

	return db.Db().Model(m).Update(attr, value).Error
}

func (m *UserDetail) TxUpdate(tx *gorm.DB, attr string, value interface{}) error {
	if m.ID == 0 {
		return fmt.Errorf("empty id")
	}

	return tx.Model(m).Update(attr, value).Error
}
