// +groupName=backup.mybackup.com
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupTargetType 备份目标类型
type BackupTargetType string

const (
	// BackupTargetTypeConfig ConfigMap和Secret备份
	BackupTargetTypeConfig BackupTargetType = "config"
	// BackupTargetTypeDatabase 数据库备份
	BackupTargetTypeDatabase BackupTargetType = "database"
	// BackupTargetTypeDirectory 系统目录备份
	BackupTargetTypeDirectory BackupTargetType = "directory"
	// BackupTargetTypeNFS NFS备份
	BackupTargetTypeNFS BackupTargetType = "nfs"
)

// BackupPhase 备份阶段
type BackupPhase string

const (
	// BackupPhasePending 备份任务创建
	BackupPhasePending BackupPhase = "Pending"
	// BackupPhaseRunning 备份任务运行中
	BackupPhaseRunning BackupPhase = "Running"
	// BackupPhaseCompleted 备份成功
	BackupPhaseCompleted BackupPhase = "Completed"
	// BackupPhaseFailed 备份失败
	BackupPhaseFailed BackupPhase = "Failed"
	// BackupPhaseExpired 备份过期
	BackupPhaseExpired BackupPhase = "Expired"
)

// RemoteStorageType 远程存储类型
type RemoteStorageType string

const (
	// RemoteStorageTypeS3 S3存储
	RemoteStorageTypeS3 RemoteStorageType = "s3"
	// RemoteStorageTypeSFTP SFTP存储
	RemoteStorageTypeSFTP RemoteStorageType = "sftp"
	// RemoteStorageTypeNFS NFS存储
	RemoteStorageTypeNFS RemoteStorageType = "nfs"
	// RemoteStorageTypeSMB SMB存储
	RemoteStorageTypeSMB RemoteStorageType = "smb"
)

// BackupTaskSpec 定义备份任务的期望状态
// +k8s:deepcopy-gen=true
type BackupTaskSpec struct {
	// 备份计划，Cron表达式
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$`
	Schedule string `json:"schedule"`

	// 备份保留策略
	// +optional
	Retention *RetentionPolicy `json:"retention,omitempty"`

	// 备份目标列表
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Targets []BackupTarget `json:"targets"`

	// 远程存储配置
	// +kubebuilder:validation:Required
	RemoteStorage RemoteStorageSpec `json:"remoteStorage"`

	// 加密配置
	// +optional
	Encryption *EncryptionSpec `json:"encryption,omitempty"`

	// Pod模板，用于自定义备份Pod的配置
	// +optional
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// 成功后的钩子
	// +optional
	PostBackupHooks []HookSpec `json:"postBackupHooks,omitempty"`

	// 资源限制
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="backup-agent:latest"
	BackupImage string `json:"backupImage,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/backup/scripts/backup.sh"
	BackupCmd string `json:"backupCmd,omitempty"`
}

// RetentionPolicy 保留策略
// +k8s:deepcopy-gen=true
type RetentionPolicy struct {
	// 保留天数
	// +optional
	MaxAgeDays *int32 `json:"maxAgeDays,omitempty"`

	// 保留备份数量
	// +optional
	MaxBackupCount *int32 `json:"maxBackupCount,omitempty"`

	// 保留最近的每小时/每天/每周备份
	// +optional
	KeepLast *KeepLastPolicy `json:"keepLast,omitempty"`
}

// KeepLastPolicy 定义保留策略
// +k8s:deepcopy-gen=true
type KeepLastPolicy struct {
	Hours  int32 `json:"hours,omitempty"`
	Days   int32 `json:"days,omitempty"`
	Weeks  int32 `json:"weeks,omitempty"`
	Months int32 `json:"months,omitempty"`
}

// BackupTarget 备份目标
// +k8s:deepcopy-gen=true
type BackupTarget struct {
	// 目标名称
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// 目标类型
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=config;database;directory;nfs
	Type BackupTargetType `json:"type"`

	// 数据库备份配置
	// +optional
	Database *DatabaseBackupSpec `json:"database,omitempty"`

	// 配置资源备份
	// +optional
	Config *ConfigBackupSpec `json:"config,omitempty"`

	// 目录备份配置
	// +optional
	Directory *DirectoryBackupSpec `json:"directory,omitempty"`

	// NFS备份配置
	// +optional
	NFS *NFSBackupSpec `json:"nfs,omitempty"`
}

// DatabaseBackupSpec 数据库备份配置
// +k8s:deepcopy-gen=true
type DatabaseBackupSpec struct {
	// 数据库类型
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=mysql;postgresql;mongodb;redis
	Type string `json:"type"`

	// 数据库地址
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// 数据库端口
	// +optional
	Port *int32 `json:"port,omitempty"`

	// 数据库名称
	// +optional
	Database string `json:"database,omitempty"`

	// 认证Secret引用
	// +kubebuilder:validation:Required
	AuthSecretRef corev1.SecretReference `json:"authSecretRef"`

	// 额外参数
	// +optional
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// 启用点时间恢复
	// +optional
	EnablePITR bool `json:"enablePITR,omitempty"`
}

// ConfigBackupSpec 配置备份配置
// +k8s:deepcopy-gen=true
type ConfigBackupSpec struct {
	// 包含的命名空间（空表示所有）
	// +optional
	IncludeNamespaces []string `json:"includeNamespaces,omitempty"`

	// 排除的命名空间
	// +optional
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`

	// 包含的资源类型
	// +optional
	ResourceTypes []string `json:"resourceTypes,omitempty"`

	// 标签选择器
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// DirectoryBackupSpec 目录备份配置
// +k8s:deepcopy-gen=true
type DirectoryBackupSpec struct {
	// 备份的目录路径（支持PVC挂载）
	// +kubebuilder:validation:Required
	Paths []DirectoryPath `json:"paths"`

	// 排除模式
	// +optional
	ExcludePatterns []string `json:"excludePatterns,omitempty"`

	// 压缩级别（0-9）
	// +optional
	CompressionLevel *int32 `json:"compressionLevel,omitempty"`
}

// DirectoryPath 目录路径
// +k8s:deepcopy-gen=true
type DirectoryPath struct {
	// 路径
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// PVC引用（如果路径来自PVC）
	// +optional
	PVCRef *corev1.TypedLocalObjectReference `json:"pvcRef,omitempty"`

	// 子路径
	// +optional
	SubPath string `json:"subPath,omitempty"`
}

// NFSBackupSpec NFS备份配置
// +k8s:deepcopy-gen=true
type NFSBackupSpec struct {
	// NFS服务器地址
	// +kubebuilder:validation:Required
	Server string `json:"server"`

	// NFS路径
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// 挂载选项
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`

	// 子目录
	// +optional
	SubPath string `json:"subPath,omitempty"`
}

// RemoteStorageSpec 远程存储配置
// +k8s:deepcopy-gen=true
type RemoteStorageSpec struct {
	// 存储类型
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;sftp;nfs;smb
	Type RemoteStorageType `json:"type"`

	// S3配置
	// +optional
	S3 *S3StorageSpec `json:"s3,omitempty"`

	// SFTP配置
	// +optional
	SFTP *SFTPStorageSpec `json:"sftp,omitempty"`

	// NFS配置
	// +optional
	NFS *NFSStorageSpec `json:"nfs,omitempty"`

	// SMB配置
	// +optional
	SMB *SMBStorageSpec `json:"smb,omitempty"`
}

// S3StorageSpec S3存储配置
// +k8s:deepcopy-gen=true
type S3StorageSpec struct {
	// 端点URL
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// 存储桶名称
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// 存储路径前缀
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// 区域
	// +optional
	Region string `json:"region,omitempty"`

	// 认证Secret引用
	// +kubebuilder:validation:Required
	AuthSecretRef corev1.SecretReference `json:"authSecretRef"`

	// 启用TLS
	// +optional
	UseTLS *bool `json:"useTLS,omitempty"`

	// 跳过TLS验证
	// +optional
	SkipTLSVerify *bool `json:"skipTLSVerify,omitempty"`
}

// SFTPStorageSpec SFTP存储配置
// +k8s:deepcopy-gen=true
type SFTPStorageSpec struct {
	// 服务器地址
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// 端口
	// +optional
	Port *int32 `json:"port,omitempty"`

	// 远程路径
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// 认证Secret引用
	// +kubebuilder:validation:Required
	AuthSecretRef corev1.SecretReference `json:"authSecretRef"`

	// SSH密钥类型
	// +optional
	SSHKeyType string `json:"sshKeyType,omitempty"`
}

// NFSStorageSpec NFS存储配置
// +k8s:deepcopy-gen=true
type NFSStorageSpec struct {
	// 服务器地址
	// +kubebuilder:validation:Required
	Server string `json:"server"`

	// 远程路径
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// 挂载选项
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`
}

// SMBStorageSpec SMB存储配置
// +k8s:deepcopy-gen=true
type SMBStorageSpec struct {
	// 服务器地址
	// +kubebuilder:validation:Required
	Server string `json:"server"`

	// 共享名称
	// +kubebuilder:validation:Required
	Share string `json:"share"`

	// 远程路径
	// +optional
	Path string `json:"path,omitempty"`

	// 认证Secret引用
	// +kubebuilder:validation:Required
	AuthSecretRef corev1.SecretReference `json:"authSecretRef"`

	// 域
	// +optional
	Domain string `json:"domain,omitempty"`
}

// EncryptionSpec 加密配置
// +k8s:deepcopy-gen=true
type EncryptionSpec struct {
	// 启用加密
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// 加密算法
	// +kubebuilder:validation:Enum=aes-256-gcm;chacha20-poly1305
	// +optional
	Algorithm string `json:"algorithm,omitempty"`

	// 加密密钥Secret引用
	// +kubebuilder:validation:Required
	KeySecretRef corev1.SecretReference `json:"keySecretRef"`

	// 压缩后加密
	// +optional
	EncryptAfterCompress *bool `json:"encryptAfterCompress,omitempty"`
}

// PodTemplateSpec Pod模板规范
// +k8s:deepcopy-gen=true
type PodTemplateSpec struct {
	// 元数据
	// +optional
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`

	// 规格
	// +optional
	Spec *corev1.PodSpec `json:"spec,omitempty"`
}

// HookSpec 钩子规范
// +k8s:deepcopy-gen=true
type HookSpec struct {
	// 钩子类型
	// +kubebuilder:validation:Enum=exec;http
	Type string `json:"type"`

	// 执行命令
	// +optional
	Exec *ExecHook `json:"exec,omitempty"`

	// HTTP请求
	// +optional
	HTTP *HTTPHook `json:"http,omitempty"`

	// 超时时间（秒）
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

// ExecHook 执行钩子
// +k8s:deepcopy-gen=true
type ExecHook struct {
	// 命令
	Command []string `json:"command"`

	// 容器名称（默认为备份容器）
	// +optional
	Container string `json:"container,omitempty"`
}

// HTTPHook HTTP钩子
// +k8s:deepcopy-gen=true
type HTTPHook struct {
	// URL
	URL string `json:"url"`

	// 方法
	// +optional
	Method string `json:"method,omitempty"`

	// 请求头
	// +optional
	Headers []HTTPHeader `json:"headers,omitempty"`

	// 请求体
	// +optional
	Body string `json:"body,omitempty"`
}

// HTTPHeader HTTP头
// +k8s:deepcopy-gen=true
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// BackupTaskStatus 定义备份任务的观察状态
// +k8s:deepcopy-gen=true
type BackupTaskStatus struct {
	// 观察到的生成
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// 备份阶段
	// +optional
	Phase BackupPhase `json:"phase,omitempty"`

	// 最后调度时间
	// +optional
	LastScheduledTime *metav1.Time `json:"lastScheduledTime,omitempty"`

	// 最后成功时间
	// +optional
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`

	// 最后完成时间
	// +optional
	LastCompletionTime *metav1.Time `json:"lastCompletionTime,omitempty"`

	// 活跃的备份作业
	// +optional
	ActiveBackupJob *corev1.ObjectReference `json:"activeBackupJob,omitempty"`

	// 备份历史记录数量
	// +optional
	BackupHistoryCount int32 `json:"backupHistoryCount,omitempty"`

	// 条件
	// +optional
	Conditions []BackupTaskCondition `json:"conditions,omitempty"`

	// 下一个调度时间
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`
}

// BackupTaskCondition 定义备份任务的条件
// +k8s:deepcopy-gen=true
type BackupTaskCondition struct {
	// 类型
	Type BackupTaskConditionType `json:"type"`

	// 状态
	Status corev1.ConditionStatus `json:"status"`

	// 最后变化时间
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// 原因
	// +optional
	Reason string `json:"reason,omitempty"`

	// 消息
	// +optional
	Message string `json:"message,omitempty"`
}

// BackupTaskConditionType 备份任务条件类型
type BackupTaskConditionType string

const (
	// BackupTaskScheduled 备份任务已调度
	BackupTaskScheduled BackupTaskConditionType = "Scheduled"
	// BackupTaskRunning 备份任务运行中
	BackupTaskRunning BackupTaskConditionType = "Running"
	// BackupTaskCompleted 备份任务完成
	BackupTaskCompleted BackupTaskConditionType = "Completed"
	// BackupTaskFailed 备份任务失败
	BackupTaskFailed BackupTaskConditionType = "Failed"
	// BackupTaskRetentionApplied 保留策略已应用
	BackupTaskRetentionApplied BackupTaskConditionType = "RetentionApplied"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bt;backuptask
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Last Success",type="date",JSONPath=".status.lastSuccessfulTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BackupTask 是备份任务的Schema
type BackupTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupTaskSpec   `json:"spec,omitempty"`
	Status BackupTaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupTaskList 包含BackupTask列表
type BackupTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupTask{}, &BackupTaskList{})
	// 如果还有其他类型，也需要注册
	SchemeBuilder.Register(&BackupRecord{}, &BackupRecordList{})
}
