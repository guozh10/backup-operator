package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupRecordSpec 定义备份记录的期望状态
// +k8s:deepcopy-gen=true
type BackupRecordSpec struct {
	// 关联的BackupTask
	// +kubebuilder:validation:Required
	BackupTaskRef corev1.ObjectReference `json:"backupTaskRef"`

	// 备份开始时间
	// +kubebuilder:validation:Required
	StartTime metav1.Time `json:"startTime"`

	// 备份完成时间
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// 备份文件信息
	// +optional
	BackupFiles []BackupFileInfo `json:"backupFiles,omitempty"`

	// 备份统计信息
	// +optional
	Statistics *BackupStatistics `json:"statistics,omitempty"`

	// 标签用于筛选
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// BackupFileInfo 备份文件信息
// +k8s:deepcopy-gen=true
type BackupFileInfo struct {
	// 文件路径
	Path string `json:"path"`

	// 文件大小（字节）
	Size int64 `json:"size"`

	// 校验和
	// +optional
	Checksum string `json:"checksum,omitempty"`

	// 存储类型
	StorageType RemoteStorageType `json:"storageType"`
}

// BackupStatistics 备份统计信息
// +k8s:deepcopy-gen=true
type BackupStatistics struct {
	// 总文件大小（字节）
	TotalSize int64 `json:"totalSize"`

	// 文件数量
	FileCount int32 `json:"fileCount"`

	// 压缩率，使用字符串表示浮点数
	// +optional
	CompressionRatio string `json:"compressionRatio,omitempty"`

	// 备份耗时（秒），使用字符串表示浮点数
	// +optional
	DurationSeconds string `json:"durationSeconds,omitempty"`
}

// BackupRecordStatus 定义备份记录的观察状态
// +k8s:deepcopy-gen=true
type BackupRecordStatus struct {
	// 备份状态
	// +optional
	Phase BackupPhase `json:"phase,omitempty"`

	// 错误信息
	// +optional
	Error string `json:"error,omitempty"`

	// 验证状态
	// +optional
	VerificationStatus VerificationStatus `json:"verificationStatus,omitempty"`

	// 过期时间
	// +optional
	ExpirationTime *metav1.Time `json:"expirationTime,omitempty"`
}

// VerificationStatus 验证状态
// +k8s:deepcopy-gen=true
type VerificationStatus struct {
	// 验证时间
	// +optional
	VerifiedAt *metav1.Time `json:"verifiedAt,omitempty"`

	// 验证结果
	// +optional
	Result VerificationResult `json:"result,omitempty"`

	// 验证消息
	// +optional
	Message string `json:"message,omitempty"`
}

// VerificationResult 验证结果
type VerificationResult string

const (
	// VerificationPending 验证待处理
	VerificationPending VerificationResult = "Pending"
	// VerificationSuccess 验证成功
	VerificationSuccess VerificationResult = "Success"
	// VerificationFailed 验证失败
	VerificationFailed VerificationResult = "Failed"
	// VerificationSkipped 验证跳过
	VerificationSkipped VerificationResult = "Skipped"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=br;backuprecord
// +kubebuilder:printcolumn:name="BackupTask",type="string",JSONPath=".spec.backupTaskRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="StartTime",type="date",JSONPath=".spec.startTime"
// +kubebuilder:printcolumn:name="Size",type="string",JSONPath=".spec.statistics.totalSize"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BackupRecord 是备份记录的Schema
type BackupRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupRecordSpec   `json:"spec,omitempty"`
	Status BackupRecordStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupRecordList 包含BackupRecord列表
type BackupRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupRecord `json:"items"`
}
