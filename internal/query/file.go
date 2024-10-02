package query

import (
	"errors"
	"fmt"
	"time"

	"github.com/Hello-Storage/hello-storage-proxy/internal/config"
	"github.com/Hello-Storage/hello-storage-proxy/internal/db"
	"github.com/Hello-Storage/hello-storage-proxy/internal/entity"
	"github.com/Hello-Storage/hello-storage-proxy/pkg/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/davecgh/go-spew/spew"
	"gorm.io/gorm"
)

// FindFileByUID returns file for the given UID.
func FindFileByUID(uid string) (*entity.File, error) {
	if uid == "" {
		return nil, fmt.Errorf("file uid required to find by uid")
	}

	var file entity.File
	err := db.Db().Where("uid = ?", uid).First(&file).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("file not found for UID: %s", uid)
		}
		return nil, fmt.Errorf("failed to find file: %v", err)
	}

	return &file, nil
}

// FileByUID returns file for the given UID.
func FindFileByID(id uint) (*entity.File, error) {
	f := &entity.File{}
	fileShareState := entity.FileShareState{}
	publicFile := entity.PublicFile{}

	err := db.Db().Model(&f).Preload("FileShareState").Where("id = ?", id).First(&f).Error
	err2 := db.Db().Where("file_uid = ?", f.UID).First(&fileShareState).Error
	err3 := db.Db().Where("file_uid = ?", f.UID).First(&publicFile).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err2 != nil && err2 != gorm.ErrRecordNotFound {
		return nil, err2
	}
	if err3 != nil && err3 != gorm.ErrRecordNotFound {
		return nil, err3
	}

	fileShareState.PublicFile = publicFile

	f.FileShareState = fileShareState

	return f, nil
}

// FindFileByCID returns file for the given CID.
func FindFileByCID(cid string) (*entity.File, error) {
	f := &entity.File{}

	err := db.Db().Where("cid = ?", cid).First(f).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	return f, nil
}

// FilesByRoot return files in a given folder root.
func FindFilesByRoot(root string) (files entity.Files, err error) {
	if err := db.Db().Where("root = ? AND deleted_at IS NULL", root).Find(&files).Error; err != nil {
		return files, err
	}

	return files, nil
}
func FindFilesByRootWithPermision(root string, userId uint) (files entity.Files, err error) {
	if err := db.Db().Table("files").Joins("INNER JOIN files_users ON files_users.file_id = files.id").Where("files.root = ? AND files.c_id_original_encrypted NOT LIKE '' AND files_users.user_id = ? AND files.deleted_at IS NULL", root, userId).Find(&files).Error; err != nil {
		return files, err
	}

	return files, nil
}

// FindSharedFilesByRoot returns shared files in a given folder root.
func FindPublicFilesByRoot(root string) (publicFiles []entity.PublicFile, err error) {
	files, err := FindFilesByRoot(root)
	if err != nil {
		return publicFiles, err
	}

	for _, file := range files {
		var publicFile entity.PublicFile

		if err := db.Db().Where("file_uid = ?",
			file.UID).First(&publicFile).Error; err != nil {
			fmt.Println(err)
		}

		publicFiles = append(publicFiles, publicFile)
	}

	return publicFiles, nil
}

func FindPublicFilesUserSharedByRoot(root string) (publicFiles []entity.PublicFileUserShared, err error) {
	files, err := FindFilesByRoot(root)
	if err != nil {
		return publicFiles, err
	}

	for _, file := range files {
		var publicFileUserShared entity.PublicFileUserShared

		if err := db.Db().Where("file_uid = ?",
			file.UID).First(&publicFileUserShared).Error; err != nil {
			fmt.Println(err)
		}

		publicFiles = append(publicFiles, publicFileUserShared)
	}

	return publicFiles, nil
}

// Count all files overall
func CountFiles() (upfile int64, err error) {
	if err := db.Db().Table("files").Count(&upfile).Error; err != nil {
		return upfile, err
	}

	return upfile, nil
}

// Count total public files in database
func CountPublicFiles() (publicfiles int64, err error) {
	if err := db.Db().Table("files").Where("encryption_status = 'public'").Count(&publicfiles).Error; err != nil {
		return publicfiles, err
	}

	return publicfiles, nil
}

// count total encrypted files in database
func CountEncryptedFiles() (encryptedfiles int64, err error) {
	if err := db.Db().Table("files").Where("encryption_status = 'encrypted'").Count(&encryptedfiles).Error; err != nil {
		return encryptedfiles, err
	}

	return encryptedfiles, nil
}

// query of the average size of all the files among the users
func CountMediumSizeFiles() (msize int64, err error) {
	if err := db.Db().Table("files").Select("ROUND(AVG(size))").Scan(&msize).Error; err != nil {
		return msize, err
	}

	return msize, nil

}

// query total sum storaged_used of all users
func CountTotalUsedStorage() (totalusedstorage int64, err error) {
	if err := db.Db().Table("user_details").Select("SUM(storage_used)").Scan(&totalusedstorage).Error; err != nil {
		return totalusedstorage, err
	}

	return totalusedstorage, nil
}

func FindShareStateByFileUID(file_uid string) (file_share_state *entity.FileShareState, file_share_states_user_share *entity.FileShareStatesUserShared, err error) {
	if err := db.Db().Preload("PublicFile").Where("file_uid = ?", file_uid).First(&file_share_state).Error; err != nil {
		if err.Error() == "record not found" {
			if err := db.Db().Preload("PublicFileUserShared").Where("file_uid = ?", file_uid).First(&file_share_states_user_share).Error; err != nil {
				return nil, nil, err
			}
			return nil, file_share_states_user_share, nil
		} else {
			return nil, nil, err
		}

	}

	return file_share_state, nil, nil
}

func CreateShareState(tx *gorm.DB, file *entity.File) (file_share_state *entity.FileShareState, err error) {
	file_share_state = &entity.FileShareState{
		FileUID: file.UID,
	}

	if err := file_share_state.TxCreate(tx); err != nil {
		return file_share_state, err
	}

	return file_share_state, nil
}

func FindUsersByFileCID(cid string) ([]uint, error) {
	var fileUsers []entity.FileUser
	var usersWF []uint

	// Join File and FileUser tables and find records by CID
	err := db.Db().
		Table("files_users").
		Select("files_users.user_id").
		Joins("JOIN files ON files.id = files_users.file_id").
		Where("files.c_id = ? AND files.deleted_at IS NULL", cid).
		Find(&fileUsers).Error

	if err != nil {
		return nil, err
	}

	// Extract user IDs from the result
	for _, fu := range fileUsers {
		usersWF = append(usersWF, fu.UserID)
	}

	return usersWF, nil
}

func FindFilesByUserAndFileCID(userID uint, cid string) ([]entity.File, error) {
	var files []entity.File

	// Join File and FileUser tables and find records by CID
	err := db.Db().Unscoped().
		Table("files_users").
		Select("files.*").
		Joins("JOIN files ON files.id = files_users.file_id").
		Where("files_users.user_id = ? AND files.c_id = ? AND files.deleted_at IS NULL", userID, cid).
		Find(&files).Error

	if err != nil {
		return nil, err
	}

	return files, nil
}

// DeleteFileByUID deletes a file by its UID.
func DeleteFileByUID(tx *gorm.DB, file_uid string) error {
	if file_uid == "" {
		return fmt.Errorf("file uid required")
	}

	// Get file_shared_state and delete it
	DeleteFileShareState(db.Db(), file_uid)

	return db.Db().Where("uid = ?", file_uid).Delete(&entity.File{}).Error
}

// query for count all txt files
func CountTxtFiles() (publicfiles int64, err error) {
	if err := db.Db().Table("files").Where("encryption_status = 'public' AND mime = 'text/plain'").Count(&publicfiles).Error; err != nil {
		return publicfiles, err
	}

	return publicfiles, nil
}

// query for count all public png files
func CountPngFiles() (publicfiles int64, err error) {
	if err := db.Db().Table("files").Where("encryption_status = 'public' AND mime = 'image/png'").Count(&publicfiles).Error; err != nil {
		return publicfiles, err
	}

	return publicfiles, nil
}

// query for count all jpg files
func CountJpgFiles() (publicfiles int64, err error) {
	if err := db.Db().Table("files").Where("encryption_status = 'public' AND (mime = 'image/jpg' OR mime = 'image/jpeg')").Count(&publicfiles).Error; err != nil {
		return publicfiles, err
	}

	return publicfiles, nil
}

// query for count all pdf files
func CountPdfFiles() (publicfiles int64, err error) {
	if err := db.Db().Table("files").Where("encryption_status = 'public' AND mime = 'application/pdf'").Count(&publicfiles).Error; err != nil {
		return publicfiles, err
	}

	return publicfiles, nil
}

// query total sum storaged_used of individual user, need id in table users for make inner join with table user_details
func CountTotalUsedStorageUser(user_uid string) (totalusedstorage int64, err error) {
	if err := db.Db().Table("user_details").Select("SUM(storage_used)").Joins("INNER JOIN users ON users.id = user_details.user_id").Where("users.uid = ? ", user_uid).Scan(&totalusedstorage).Error; err != nil {
		return totalusedstorage, err
	}

	return totalusedstorage, nil
}

// query total sum user encrypted files (encyption_status), need id in table users for make inner join with table files_users and files
func CountTotalEncryptedFilesUser(user_uid string) (encryptedfiles int64, err error) {
	if err := db.Db().Table("files").Select("COUNT(*)").Joins("INNER JOIN files_users ON files_users.file_id = files.id").Joins("INNER JOIN users ON users.id = files_users.user_id").Where("users.uid = ? AND files.encryption_status = 'encrypted'  AND files.deleted_at IS NULL", user_uid).Scan(&encryptedfiles).Error; err != nil {
		return encryptedfiles, err
	}

	return encryptedfiles, nil
}

// query total sum user publicfiles files (encyption_status), need id in table users for make inner join with table files_users and files
func CountTotalPublicFilesUser(user_uid string) (publicfiles int64, err error) {
	if err := db.Db().Table("files").Select("COUNT(*)").Joins("INNER JOIN files_users ON files_users.file_id = files.id").Joins("INNER JOIN users ON users.id = files_users.user_id").Where("users.uid = ? AND files.encryption_status = 'public' AND files.deleted_at IS NULL", user_uid).Scan(&publicfiles).Error; err != nil {
		return publicfiles, err
	}

	return publicfiles, nil
}

// query total sum user publicfiles and encryptedfiles files (encyption_status), need id in table users for make inner join with table files_users and files
func CountTotalFilesUser(user_uid string) (upfile int64, err error) {
	if err := db.Db().Table("files").Select("COUNT(*)").Joins("INNER JOIN files_users ON files_users.file_id = files.id").Joins("INNER JOIN users ON users.id = files_users.user_id").Where("users.uid = ?  AND files.deleted_at IS NULL", user_uid).Scan(&upfile).Error; err != nil {
		return upfile, err
	}

	return upfile, nil
}

// Query storage used by user up to a specific date
func CountStorageUsed(daystring string, user_uid string) (dailystorage int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(SUM(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR DATE(files.deleted_at) > DATE(?)) THEN files.size WHEN files_users.permission != 'shared' AND (files.deleted_at IS NOT NULL AND DATE(files.deleted_at) < DATE(?)) THEN -files.size ELSE 0 END), 0), 0)", daystring, daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("users.uid = ? AND DATE(files.created_at) <= DATE(?)", user_uid, daystring)

	// Execute and scan the result
	if err := query.Scan(&dailystorage).Error; err != nil {
		return 0, err
	}

	return dailystorage, nil
}

// Query storage used up to a specific date
func CountPublicStorageUsed(daystring string) (dailystorage int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(SUM(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR DATE(files.deleted_at) > DATE(?)) THEN files.size WHEN files_users.permission != 'shared' AND (files.deleted_at IS NOT NULL AND DATE(files.deleted_at) < DATE(?)) THEN -files.size ELSE 0 END), 0), 0)", daystring, daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("DATE(files.created_at) <= DATE(?)", daystring)

	// Execute and scan the result
	if err := query.Scan(&dailystorage).Error; err != nil {
		return 0, err
	}

	return dailystorage, nil
}

// Query total files used by user up to a specific date
func CountFilesUsed(daystring string, user_uid string) (dailyfiles int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(COUNT(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR DATE(files.deleted_at) > DATE(?)) THEN 1 END), 0), 0)", daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("users.uid = ? AND DATE(files.created_at) <= DATE(?)", user_uid, daystring)

	// Execute and scan the result
	if err := query.Scan(&dailyfiles).Error; err != nil {
		return 0, err
	}

	return dailyfiles, nil
}

// Query public files used by user up to a specific date
func CountFilesUsedByStatus(daystring string, user_uid string, status string) (dailypublicfiles int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(COUNT(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR DATE(files.deleted_at) > DATE(?)) THEN 1 END), 0), 0)", daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("users.uid = ? AND DATE(files.created_at) <= DATE(?) AND files.encryption_status = ?", user_uid, daystring, status)

	// Execute and scan the result
	if err := query.Scan(&dailypublicfiles).Error; err != nil {
		return 0, err
	}

	return dailypublicfiles, nil
}

// Query storage used by user up to a specific date & hour
func CountStorageUsedH(daystring string, user_uid string) (dailystorage int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(SUM(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR (files.deleted_at) > (?)) THEN files.size WHEN files_users.permission != 'shared' AND (files.deleted_at IS NOT NULL AND DATE(files.deleted_at) < DATE(?)) THEN -files.size ELSE 0 END), 0), 0)", daystring, daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("users.uid = ? AND (files.created_at) <= (?)", user_uid, daystring)

	// Execute and scan the result
	if err := query.Scan(&dailystorage).Error; err != nil {
		return 0, err
	}

	return dailystorage, nil
}

// Query total files used by user up to a specific date & hour
func CountFilesUsedH(daystring string, user_uid string) (dailyfiles int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(COUNT(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR (files.deleted_at) > (?)) THEN 1 END), 0), 0)", daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("users.uid = ? AND (files.created_at) <= (?)", user_uid, daystring)

	// Execute and scan the result
	if err := query.Scan(&dailyfiles).Error; err != nil {
		return 0, err
	}

	return dailyfiles, nil
}

// Query public files used by user up to a specific date & hour
func CountFilesUsedByStatusH(daystring string, user_uid string, status string) (dailypublicfiles int64, err error) {
	query := db.Db().
		Table("files").
		Select("GREATEST(COALESCE(COUNT(CASE WHEN files_users.permission != 'shared' AND (files.deleted_at IS NULL OR (files.deleted_at) > (?)) THEN 1 END), 0), 0)", daystring).
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Where("users.uid = ? AND (files.created_at) <= (?) AND files.encryption_status = ?", user_uid, daystring, status)

	// Execute and scan the result
	if err := query.Scan(&dailypublicfiles).Error; err != nil {
		return 0, err
	}

	return dailypublicfiles, nil
}

func GetStartAndEndFileDatesPublic() (time.Time, time.Time, error) {
	var minDate, maxDate time.Time

	// Query for the earliest creation date
	err := db.Db().
		Table("files").
		Select("MIN(files.created_at)").
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Scan(&minDate).Error
	if err != nil {
		return minDate, maxDate, err
	}

	// Query for the latest creation date
	err = db.Db().
		Table("files").
		Select("MAX(files.created_at)").
		Joins("INNER JOIN files_users ON files_users.file_id = files.id").
		Joins("INNER JOIN users ON users.id = files_users.user_id").
		Scan(&maxDate).Error
	if err != nil {
		return minDate, maxDate, err
	}

	if minDate.IsZero() || maxDate.IsZero() { // Handle the case where there are no public files
		// This could be returning an error or default dates
		return time.Time{}, time.Time{}, errors.New("no public files found")
	}

	return minDate, maxDate, nil
}

func CountTotalFiles(encryptionType string, daystring string) (totalFiles int64, err error) {
	query := db.Db().
		Table("files").
		Select("COALESCE(COUNT(*), 0)").
		Where("files.encryption_status = ? AND DATE(files.created_at) <= DATE(?)", encryptionType, daystring)

	if err := query.Scan(&totalFiles).Error; err != nil {
		return totalFiles, err
	}

	return totalFiles, nil
}

func QueryShareGroupByHash(shareGroupHash string) ([]string, error) {
	var publicFileShareGroups []entity.PublicFileShareGroup

	// Query the database to fetch records of PublicFileShareGroup associated with the shareGroupHash
	if err := db.Db().Where("share_group_hash = ?", shareGroupHash).Find(&publicFileShareGroups).Error; err != nil {
		return nil, err
	}

	// Extract share hashes from PublicFileShareGroup records
	var shareHashes []string
	for _, publicFileShareGroup := range publicFileShareGroups {
		shareHashes = append(shareHashes, publicFileShareGroup.ShareHash)
	}

	return shareHashes, nil
}

// DeletePublicFileShareGroupByShareHash deletes the record in publicFileShareGroups
// based on the sharing hash.
func DeletePublicFileShareGroupByShareHash(tx *gorm.DB, shareHash string) error {
	var publicFileShareGroup entity.PublicFileShareGroup

	// Find the record based on the sharing hash
	result := tx.Where("share_hash = ?", shareHash).First(&publicFileShareGroup)

	// Check for errors during the query
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Record not found, return without error
			return nil
		}
		// Other error occurred
		return err
	}

	// Delete the record
	if err := tx.Delete(&publicFileShareGroup).Error; err != nil {
		return err
	}

	return nil
}

func DeleteFileShareStatesByFileUID(fileUID string) error {
	return db.Db().Where("file_uid = ?", fileUID).Delete(&entity.FileShareState{}).Error
}

func DeleteEmptyShareGroup(tx *gorm.DB, shareGroupHash string) error {
	var count int64
	if err := tx.Model(&entity.PublicFileShareGroup{}).Where("share_group_hash = ?", shareGroupHash).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return tx.Unscoped().Where("hash = ?", shareGroupHash).Delete(&entity.ShareGroup{}).Error
	}

	return nil
}

func GetApiFiles(user_id uint) (files entity.Files, err error) {
	var apiFiles []entity.ApiKeyFile
	if err := db.Db().Where("user_id = ?", user_id).Find(&apiFiles).Error; err != nil {
		return nil, err
	}

	var fileIDs []uint
	for _, apiFile := range apiFiles {
		fileIDs = append(fileIDs, apiFile.FileID)
	}

	if err := db.Db().Where("id IN ?", fileIDs).Find(&files).Error; err != nil {
		return nil, err
	}

	return files, nil
}

// FindFilesNotInPool returns files that are not in the pool.
func FindFilesNotInPool() (files entity.Files, err error) {
	var allFiles []entity.File
	//get all files
	if err := db.Db().Find(&allFiles).Error; err != nil {
		return nil, err
	}

	var filesNotInPool entity.Files
	// Create a map to track CIDs
	cidChecked := make(map[string]bool)

	s3Config := aws.Config{
		Credentials: credentials.NewStaticCredentials(
			config.Env().StorageAccessKey,
			config.Env().StorageSecretKey,
			"",
		),
		Endpoint:         aws.String(config.Env().StorageEndpoint),
		Region:           aws.String(config.Env().StorageRegion),
		S3ForcePathStyle: aws.Bool(true),
	}

	totalFiles := len(allFiles)
	log.Printf("Total files: %d", totalFiles)

	for i, file := range allFiles {
		if _, checked := cidChecked[file.CID]; !checked {
			_, err := s3.HeadObject(s3Config, config.Env().StorageBucket, file.CID)
			cidChecked[file.CID] = (err == nil) // true if exists in S3, false otherwise

			if i%100 == 0 || i == totalFiles-1 {
				log.Printf("S3 Check Progress: %d of %d files checked", i+1, totalFiles)

			}
		}
	}

	log.Printf("S3 check completed. %d files checked.", totalFiles)

	// Create a list of files that are not in the pool
	for _, file := range allFiles {
		if !cidChecked[file.CID] {
			filesNotInPool = append(filesNotInPool, file)
		}
	}

	return filesNotInPool, nil
}

// get if file is in a shared folder or not
func IsInSharedFolder(fileRoot string, userID uint) bool {

	// if file root is empty or user id is 0, return false
	if fileRoot == "" || userID == 0 || fileRoot == "/" {
		return false
	}

	// get folder id by file root (folder uid)
	query := db.Db().
		Table("folders").
		Select("id").
		Where("uid=?", fileRoot)

	var folderID uint

	if err := query.Scan(&folderID).Error; err != nil {
		return false
	}

	// get folder user by folder id and user id
	query = db.Db().Table("folder_users").Select("*").
		Where("user_id = ? AND folder_id = ?", userID, folderID)

	var folderUser entity.FolderUser

	if err := query.Scan(&folderUser).Error; err != nil {
		return false
	}

	spew.Dump(folderUser)

	// if folder user is shared, return true
	if folderUser.Permission == "shared" {
		return true
	}

	// else return false
	return false
}
