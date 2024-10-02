package entity

import (
	"time"

	"github.com/Hello-Storage/hello-storage-proxy/internal/db"
	"gorm.io/gorm"
)

type PublicFile struct {
	ID                   uint           `gorm:"primarykey"                   json:"id"`
	FileUID              string         `gorm:"type:varchar(42);uniqueIndex" json:"file_uid"`
	ShareHash            string         `gorm:"type:varchar(256)"            json:"share_hash"`
	Name                 string         `gorm:"type:varchar(1024);"          json:"name"`
	Mime                 string         `gorm:"type:varchar(256)"            json:"mime_type"`
	Size                 int64          `                                   json:"size"`
	CID                  string         `gorm:"type:varchar(64)"             json:"cid"`
	CIDOriginalDecrypted string         `gorm:"type:varchar(256)"            json:"cid_original_decrypted"`
	CreatedAt            time.Time      `gorm:"index"                        json:"created_at"`
	UpdatedAt            time.Time      `gorm:"index"                        json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index"                        json:"deleted_at"`
	HasBeenOpened        *bool          `json:"has_been_opened" gorm:"default:NULL"`
	ExpireAt             *time.Time     `json:"expire_at" gorm:"default:NULL"`
}

func (PublicFile) TableName() string {
	return "public_files"
}

func (m *PublicFile) Create() error {
	return db.Db().Create(m).Error
}

func (m *PublicFile) TxCreate(tx *gorm.DB) error {
	return tx.Create(m).Error
}

func (m *PublicFile) Save() error {
	return db.Db().Save(m).Error
}

func (m *PublicFile) TxSave(tx *gorm.DB) error {
	return tx.Save(m).Error
}

func (m *PublicFile) Delete() error {
	return db.Db().Unscoped().Delete(m).Error
}

func (m *PublicFile) TxDelete(tx *gorm.DB) error {
	return tx.Unscoped().Delete(m).Error
}
