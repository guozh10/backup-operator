package controller

// 控制器常量
const (
	// 备份任务Finalizer
	backupTaskFinalizer = "backup.mybackup.com/finalizer"

	// 标签键
	backupTaskLabelKey    = "backup.mybackup.com/backup-task"
	backupTaskUIDLabelKey = "backup.mybackup.com/backup-task-uid"
	createdByLabelKey     = "backup.mybackup.com/created-by"

	// 注解键
	scheduleAnnotationKey   = "backup.mybackup.com/schedule"
	lastBackupAnnotationKey = "backup.mybackup.com/last-backup"

	// 容器常量
	backupContainerName = "backup"
	// backupImage         = "easzlab.io.local/backup-agent:latest"
	defaultBackupImage = "backup-agent:latest"
	defaultBackupCmd   = "/backup/scripts/backup.sh"
	// 卷名称
	backupDataVolumeName = "backup-data"
	scriptsVolumeName    = "backup-scripts"
	configVolumeName     = "backup-config"
	temporaryVolumeName  = "temp"
)
