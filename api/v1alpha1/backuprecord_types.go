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
)
// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupRecordSpec defines the desired state of BackupRecord
type BackupRecordSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of BackupRecord. Edit backuprecord_types.go to remove/update
	// +optional
	// Foo *string `json:"foo,omitempty"`
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

// 备份文件信息
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

// 备份统计信息
type BackupStatistics struct {
	// 总文件大小（字节）
	TotalSize int64 `json:"totalSize"`

	// 文件数量
	FileCount int32 `json:"fileCount"`

	// 压缩率
	// +optional
	CompressionRatio string `json:"compressionRatio,omitempty"`

	// 备份耗时（秒）
	// +optional
	DurationSeconds string `json:"durationSeconds,omitempty"`
}

// BackupRecordStatus定义备份记录的观察状态
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

// 验证状态
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

// 验证结果
type VerificationResult string

const (
	VerificationPending   VerificationResult = "Pending"
	VerificationSuccess   VerificationResult = "Success"
	VerificationFailed    VerificationResult = "Failed"
	VerificationSkipped   VerificationResult = "Skipped"
)

// BackupRecord是备份记录的Schema
type BackupRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupRecordSpec   `json:"spec,omitempty"`
	Status BackupRecordStatus `json:"status,omitempty"`
}

// BackupRecordList包含BackupRecord列表
type BackupRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupRecord{}, &BackupRecordList{})
}


// BackupRecordStatus defines the observed state of BackupRecord.
// type BackupRecordStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the BackupRecord resource.
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
// }

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BackupRecord is the Schema for the backuprecords API
// type BackupRecord struct {
// 	metav1.TypeMeta `json:",inline"`

// 	// metadata is a standard object metadata
// 	// +optional
// 	metav1.ObjectMeta `json:"metadata,omitzero"`

// 	// spec defines the desired state of BackupRecord
// 	// +required
// 	Spec BackupRecordSpec `json:"spec"`

// 	// status defines the observed state of BackupRecord
// 	// +optional
// 	Status BackupRecordStatus `json:"status,omitzero"`
// }

// // +kubebuilder:object:root=true

// // BackupRecordList contains a list of BackupRecord
// type BackupRecordList struct {
// 	metav1.TypeMeta `json:",inline"`
// 	metav1.ListMeta `json:"metadata,omitzero"`
// 	Items           []BackupRecord `json:"items"`
// }

// func init() {
// 	SchemeBuilder.Register(&BackupRecord{}, &BackupRecordList{})
// }
