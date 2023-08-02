package main

type Tab struct {
	URL           string `json:"url"`
	Title         string `json:"title"`
	ID            int    `json:"id"`
	OpenerTabID   int    `json:"openerTabId,omitempty"`
	CookieStoreID string `json:"cookieStoreId,omitempty"`
}

type STGPayload struct {
	Version string `json:"version"`
	Groups  []struct {
		ID                                       int           `json:"id"`
		Title                                    string        `json:"title"`
		IconColor                                string        `json:"iconColor"`
		IconURL                                  interface{}   `json:"iconUrl"`
		IconViewType                             string        `json:"iconViewType"`
		Tabs                                     []Tab         `json:"tabs"`
		IsArchive                                bool          `json:"isArchive"`
		NewTabContainer                          string        `json:"newTabContainer"`
		IfDifferentContainerReOpen               bool          `json:"ifDifferentContainerReOpen"`
		ExcludeContainersForReOpen               []interface{} `json:"excludeContainersForReOpen"`
		IsMain                                   bool          `json:"isMain"`
		IsSticky                                 bool          `json:"isSticky"`
		CatchTabContainers                       []interface{} `json:"catchTabContainers"`
		CatchTabRules                            string        `json:"catchTabRules"`
		MoveToMainIfNotInCatchTabRules           bool          `json:"moveToMainIfNotInCatchTabRules"`
		MuteTabsWhenGroupCloseAndRestoreWhenOpen bool          `json:"muteTabsWhenGroupCloseAndRestoreWhenOpen"`
		ShowTabAfterMovingItIntoThisGroup        bool          `json:"showTabAfterMovingItIntoThisGroup"`
		DontDiscardTabsAfterHideThisGroup        bool          `json:"dontDiscardTabsAfterHideThisGroup"`
		BookmarkID                               interface{}   `json:"bookmarkId"`
	} `json:"groups"`
	LastCreatedGroupPosition              int      `json:"lastCreatedGroupPosition"`
	DiscardTabsAfterHide                  bool     `json:"discardTabsAfterHide"`
	DiscardAfterHideExcludeAudioTabs      bool     `json:"discardAfterHideExcludeAudioTabs"`
	ClosePopupAfterChangeGroup            bool     `json:"closePopupAfterChangeGroup"`
	OpenGroupAfterChange                  bool     `json:"openGroupAfterChange"`
	AlwaysAskNewGroupName                 bool     `json:"alwaysAskNewGroupName"`
	PrependGroupTitleToWindowTitle        bool     `json:"prependGroupTitleToWindowTitle"`
	CreateNewGroupWhenOpenNewWindow       bool     `json:"createNewGroupWhenOpenNewWindow"`
	ShowNotificationAfterMoveTab          bool     `json:"showNotificationAfterMoveTab"`
	OpenManageGroupsInTab                 bool     `json:"openManageGroupsInTab"`
	ShowConfirmDialogBeforeGroupArchiving bool     `json:"showConfirmDialogBeforeGroupArchiving"`
	ShowConfirmDialogBeforeGroupDelete    bool     `json:"showConfirmDialogBeforeGroupDelete"`
	ShowNotificationAfterGroupDelete      bool     `json:"showNotificationAfterGroupDelete"`
	ShowContextMenuOnTabs                 bool     `json:"showContextMenuOnTabs"`
	ShowContextMenuOnLinks                bool     `json:"showContextMenuOnLinks"`
	ExportGroupToMainBookmarkFolder       bool     `json:"exportGroupToMainBookmarkFolder"`
	DefaultBookmarksParent                string   `json:"defaultBookmarksParent"`
	LeaveBookmarksOfClosedTabs            bool     `json:"leaveBookmarksOfClosedTabs"`
	ShowExtendGroupsPopupWithActiveTabs   bool     `json:"showExtendGroupsPopupWithActiveTabs"`
	ShowTabsWithThumbnailsInManageGroups  bool     `json:"showTabsWithThumbnailsInManageGroups"`
	FullPopupWidth                        bool     `json:"fullPopupWidth"`
	TemporaryContainerTitle               string   `json:"temporaryContainerTitle"`
	ContextMenuTab                        []string `json:"contextMenuTab"`
	ContextMenuGroup                      []string `json:"contextMenuGroup"`
	DefaultGroupIconViewType              string   `json:"defaultGroupIconViewType"`
	DefaultGroupIconColor                 string   `json:"defaultGroupIconColor"`
	AutoBackupEnable                      bool     `json:"autoBackupEnable"`
	AutoBackupLastBackupTimeStamp         int      `json:"autoBackupLastBackupTimeStamp"`
	AutoBackupIntervalKey                 string   `json:"autoBackupIntervalKey"`
	AutoBackupIntervalValue               int      `json:"autoBackupIntervalValue"`
	AutoBackupIncludeTabThumbnails        bool     `json:"autoBackupIncludeTabThumbnails"`
	AutoBackupIncludeTabFavIcons          bool     `json:"autoBackupIncludeTabFavIcons"`
	AutoBackupGroupsToBookmarks           bool     `json:"autoBackupGroupsToBookmarks"`
	AutoBackupGroupsToFile                bool     `json:"autoBackupGroupsToFile"`
	AutoBackupFolderName                  string   `json:"autoBackupFolderName"`
	AutoBackupByDayIndex                  bool     `json:"autoBackupByDayIndex"`
	Theme                                 string   `json:"theme"`
	Hotkeys                               []struct {
		CtrlKey  bool   `json:"ctrlKey"`
		ShiftKey bool   `json:"shiftKey"`
		AltKey   bool   `json:"altKey"`
		MetaKey  bool   `json:"metaKey"`
		Key      string `json:"key"`
		KeyCode  int    `json:"keyCode"`
		Action   string `json:"action"`
		GroupID  int    `json:"groupId"`
	} `json:"hotkeys"`
	PinnedTabs []struct {
		URL   string `json:"url"`
		Title string `json:"title"`
		ID    int    `json:"id"`
	} `json:"pinnedTabs"`
	Containers struct {
		FirefoxContainer7 struct {
			Name          string `json:"name"`
			Icon          string `json:"icon"`
			IconURL       string `json:"iconUrl"`
			Color         string `json:"color"`
			ColorCode     string `json:"colorCode"`
			CookieStoreID string `json:"cookieStoreId"`
		} `json:"firefox-container-7"`
		FirefoxContainer12 struct {
			Name          string `json:"name"`
			Icon          string `json:"icon"`
			IconURL       string `json:"iconUrl"`
			Color         string `json:"color"`
			ColorCode     string `json:"colorCode"`
			CookieStoreID string `json:"cookieStoreId"`
		} `json:"firefox-container-12"`
	} `json:"containers"`
}
