package entity

import (
	"time"

	"github.com/Hello-Storage/hello-storage-proxy/internal/db"
	"github.com/Hello-Storage/hello-storage-proxy/pkg/rnd"
	"gorm.io/gorm"
)

const (
	FolderUID = byte('d')
)

type Folders []Folder

type Folder struct {
	ID               uint             `gorm:"primarykey"                          json:"id"`
	UID              string           `gorm:"type:varchar(42);uniqueIndex;"       json:"uid"`
	CID              string           `gorm:"type:varchar(64)" json:"cid"`
	Title            string           `gorm:"type:varchar(255);"                  json:"title"`
	Path             string           `gorm:"type:varchar(1024);default:'/';"     json:"path"` // folderA/folderB/***
	Root             string           `gorm:"type:varchar(42);index;default:'/';" json:"root"` // parent folder uid
	CreatedAt        time.Time        `                                           json:"created_at"`
	UpdatedAt        time.Time        `                                           json:"updated_at"`
	DeletedAt        gorm.DeletedAt   `gorm:"index"                               json:"deleted_at"`
	IsInPool         bool             `gorm:"type:boolean;default:false;"         json:"is_in_pool"`
	EncryptionStatus EncryptionStatus `gorm:"type:encryption_status;default:'public'" json:"encryption_status"`
}

// TableName returns the entity table name.
func (Folder) TableName() string {
	return "folders"
}

func (m *Folder) Create() error {
	return db.Db().Create(m).Error
}

func (m *Folder) TxCreate(tx *gorm.DB) error {
	return tx.Create(m).Error
}

// BeforeCreate creates a random UID if needed before inserting a new row to the database.
func (m *Folder) BeforeCreate(db *gorm.DB) error {
	if rnd.IsUnique(m.UID, 'd') {
		return nil
	}
	m.UID = rnd.GenerateUID(FolderUID)
	db.Statement.SetColumn("UID", m.UID)

	return nil
}

func (m *Folder) FirstOrCreateFolderByTitleAndRoot() *Folder {
	result := Folder{}

	if err := db.Db().Where("title = ? AND root = ?", m.Title, m.Root).First(&result).Error; err == nil {
		return &result
	} else if err := m.Create(); err != nil {
		log.Errorf("Folder first or create: %s", err)
		return nil
	}

	return m
}

// update
func (m *Folder) UpdateRootOnly() error {
	return db.Db().Model(m).Where("UID = ?", m.UID).Update("Root", m.Root).Error

}

// IsFolderOwner checks if a user is the owner of a folder
func IsFolderOwner(folderID uint, userID uint) (bool, error) {
	var count int64
	err := db.Db().Table("folders_users").
		Where("folder_id = ? AND user_id = ? AND permission = ?", folderID, userID, OwnerPermission).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateTitle updates the folder title with the new title provided.
func (m *Folder) UpdateTitle(newTitle string) error {

	if err := db.Db().Model(m).Where("UID = ?", m.UID).Update("Title", newTitle).Error; err != nil {
		return err
	}

	return nil
}

// UpdateEncryptionStatus updates the EncryptionStatus for the folder.
func (m *Folder) UpdateEncryptionStatus(newEncryptionStatus EncryptionStatus) error {

	if err := db.Db().Model(m).Where("UID = ?", m.UID).Update("EncryptionStatus", newEncryptionStatus).Error; err != nil {
		return err
	}

	return nil
}

// UpdateEncryptionStatusAndCID updates the EncryptionStatus for the folder and the cid.
func (m *Folder) UpdateEncryptionStatusAndCID(newEncryptionStatus EncryptionStatus, userID string) error {

	if err := db.Db().Model(m).Where("uid = ?", m.UID).Update("EncryptionStatus", newEncryptionStatus).Error; err != nil {
		return err
	}
	if err := db.Db().Model(m).Where("uid = ?", m.UID).Update("c_id", userID+m.UID).Error; err != nil {
		return err
	}

	return nil
}
