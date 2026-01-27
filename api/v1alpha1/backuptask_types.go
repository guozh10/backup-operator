/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// 备份目标类型
type BackupTargetType string
const (
	// ConfigMap和Secret备份
	BackupTargetTypeConfig BackupTargetType = "config"
	// 数据库备份
	BackupTargetTypeDatabase BackupTargetType = "database"
	// 系统目录备份
	BackupTargetTypeDirectory BackupTargetType = "directory"
	// NFS备份
	BackupTargetTypeNFS BackupTargetType = "nfs"
)
// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// 备份阶段
type BackupPhase string

const (
	// 备份任务创建
	BackupPhasePending BackupPhase = "Pending"
	// 备份任务运行中
	BackupPhaseRunning BackupPhase = "Running"
	// 备份成功
	BackupPhaseCompleted BackupPhase = "Completed"
	// 备份失败
	BackupPhaseFailed BackupPhase = "Failed"
	// 备份过期
	BackupPhaseExpired BackupPhase = "Expired"
)

// 远程存储类型
type RemoteStorageType string

const (
	RemoteStorageTypeS3   RemoteStorageType = "s3"
	RemoteStorageTypeSFTP RemoteStorageType = "sftp"
	RemoteStorageTypeNFS  RemoteStorageType = "nfs"
	RemoteStorageTypeSMB  RemoteStorageType = "smb"
)

// BackupTaskSpec defines the desired state of BackupTask
type BackupTaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of BackupTask. Edit backuptask_types.go to remove/update
	// +optional
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
	//Foo *string `json:"foo,omitempty"`
}
// 保留策略
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
// KeepLastPolicy定义保留策略
type KeepLastPolicy struct {
	Hours   int32 `json:"hours,omitempty"`
	Days    int32 `json:"days,omitempty"`
	Weeks   int32 `json:"weeks,omitempty"`
	Months  int32 `json:"months,omitempty"`
}

// 备份目标
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

// 数据库备份配置
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

// 配置备份配置
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

// 目录备份配置
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

// 目录路径
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

// NFS备份配置
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

// 远程存储配置
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

// S3存储配置
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

// SFTP存储配置
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

// NFS存储配置
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

// SMB存储配置
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

// 加密配置
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

// Pod模板规范
type PodTemplateSpec struct {
	// 元数据
	// +optional
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`

	// 规格
	// +optional
	Spec *corev1.PodSpec `json:"spec,omitempty"`
}

// 钩子规范
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

// 执行钩子
type ExecHook struct {
	// 命令
	Command []string `json:"command"`

	// 容器名称（默认为备份容器）
	// +optional
	Container string `json:"container,omitempty"`
}

// HTTP钩子
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

// HTTP头
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// BackupTaskStatus defines the observed state of BackupTask.
type BackupTaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the BackupTask resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	// Conditions []metav1.Condition `json:"conditions,omitempty"`
	// 观察到的生成
	// +optional
	ObservedGeneration []int64 `json:"observedGeneration,omitempty"`

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

// BackupTaskCondition定义备份任务的条件
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

// 备份任务条件类型
type BackupTaskConditionType string

const (
	// 备份任务已调度
	BackupTaskScheduled BackupTaskConditionType = "Scheduled"
	// 备份任务运行中
	BackupTaskRunning BackupTaskConditionType = "Running"
	// 备份任务完成
	BackupTaskCompleted BackupTaskConditionType = "Completed"
	// 备份任务失败
	BackupTaskFailed BackupTaskConditionType = "Failed"
	// 保留策略已应用
	BackupTaskRetentionApplied BackupTaskConditionType = "RetentionApplied"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bt
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Last Success",type="date",JSONPath=".status.lastSuccessfulTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BackupTask is the Schema for the backuptasks API
type BackupTask struct {
	// metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	// metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of BackupTask
	// +required
	// Spec BackupTaskSpec `json:"spec"`

	// status defines the observed state of BackupTask
	// +optional
	// Status BackupTaskStatus `json:"status,omitzero"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupTaskSpec   `json:"spec,omitempty"`
	Status BackupTaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupTaskList contains a list of BackupTask
type BackupTaskList struct {
	// metav1.TypeMeta `json:",inline"`
	// metav1.ListMeta `json:"metadata,omitzero"`
	// Items           []BackupTask `json:"items"`
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupTask{}, &BackupTaskList{})
}
