package ellimango

type Folder struct {
	Oracle Oracle
	Redis  Redis
}

// get listing folder id by user id and folder name
func (folder *Folder) GetListingFolderIdByUserIdAndFolderName(userId string, folderName string) (string, string) {
	folderId, errorMessage := folder.Oracle.GetListingFolderIdByUserIdAndFolderName(userId, folderName)
	return folderId, errorMessage
}

// create listing folder with user id and folder name
func (folder *Folder) CreateListingFolderWithUserIdAndFolderName(userId string, folderName string) (uint64, string) {
	folderId, errorMessage := folder.Oracle.CreateListingFolderWithUserIdAndFolderName(userId, folderName)
	return folderId, errorMessage
}

// get listing folder name by folder id
func (folder *Folder) GetListingFolderNameByFolderId(folderId string) (string, string) {
	folderName, errorMessage := folder.Oracle.GetListingFolderNameByFolderId(folderId)
	return folderName, errorMessage
}
