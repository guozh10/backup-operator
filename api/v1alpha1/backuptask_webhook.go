package v1alpha1

import (
	"context"
	"fmt"
	"regexp"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ webhook.Defaulter = &BackupTask{}
var _ webhook.Validator = &BackupTask{}

// Default 实现默认值设置
func (r *BackupTask) Default() {
	if r.Spec.PodTemplate == nil {
		r.Spec.PodTemplate = &PodTemplateSpec{}
	}
	
	if r.Spec.Encryption != nil && r.Spec.Encryption.Enabled && r.Spec.Encryption.Algorithm == "" {
		r.Spec.Encryption.Algorithm = "aes-256-gcm"
	}
	
	if r.Spec.Encryption != nil && r.Spec.Encryption.EncryptAfterCompress == nil {
		encryptAfterCompress := true
		r.Spec.Encryption.EncryptAfterCompress = &encryptAfterCompress
	}
}

// ValidateCreate 实现创建时的验证
func (r *BackupTask) ValidateCreate() (admission.Warnings, error) {
	return nil, r.validateBackupTask()
}

// ValidateUpdate 实现更新时的验证
func (r *BackupTask) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	oldBackupTask, ok := old.(*BackupTask)
	if !ok {
		return nil, apierrors.NewBadRequest("expected a BackupTask object")
	}
	
	// 不允许修改schedule
	if r.Spec.Schedule != oldBackupTask.Spec.Schedule {
		return nil, field.Forbidden(
			field.NewPath("spec", "schedule"),
			"schedule cannot be modified",
		).ToAggregate()
	}
	
	return nil, r.validateBackupTask()
}

// ValidateDelete 实现删除时的验证
func (r *BackupTask) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (r *BackupTask) validateBackupTask() error {
	var allErrs field.ErrorList
	
	// 验证schedule格式
	if !isValidCronSchedule(r.Spec.Schedule) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "schedule"),
			r.Spec.Schedule,
			"invalid cron schedule format",
		))
	}
	
	// 验证至少有一个目标
	if len(r.Spec.Targets) == 0 {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec", "targets"),
			"at least one backup target is required",
		))
	}
	
	// 验证每个目标
	for i, target := range r.Spec.Targets {
		targetPath := field.NewPath("spec", "targets").Index(i)
		
		// 验证目标名称
		if target.Name == "" {
			allErrs = append(allErrs, field.Required(
				targetPath.Child("name"),
				"target name is required",
			))
		}
		
		// 根据类型验证具体配置
		switch target.Type {
		case BackupTargetTypeDatabase:
			if target.Database == nil {
				allErrs = append(allErrs, field.Required(
					targetPath.Child("database"),
					"database configuration is required for database backup",
				))
			} else {
				allErrs = append(allErrs, r.validateDatabaseSpec(targetPath.Child("database"), target.Database)...)
			}
			
		case BackupTargetTypeConfig:
			if target.Config == nil {
				allErrs = append(allErrs, field.Required(
					targetPath.Child("config"),
					"config configuration is required for config backup",
				))
			}
			
		case BackupTargetTypeDirectory:
			if target.Directory == nil {
				allErrs = append(allErrs, field.Required(
					targetPath.Child("directory"),
					"directory configuration is required for directory backup",
				))
			} else {
				allErrs = append(allErrs, r.validateDirectorySpec(targetPath.Child("directory"), target.Directory)...)
			}
			
		case BackupTargetTypeNFS:
			if target.NFS == nil {
				allErrs = append(allErrs, field.Required(
					targetPath.Child("nfs"),
					"nfs configuration is required for nfs backup",
				))
			} else {
				allErrs = append(allErrs, r.validateNFSSpec(targetPath.Child("nfs"), target.NFS)...)
			}
		}
	}
	
	// 验证远程存储配置
	allErrs = append(allErrs, r.validateRemoteStorage(field.NewPath("spec", "remoteStorage"), &r.Spec.RemoteStorage)...)
	
	// 验证加密配置
	if r.Spec.Encryption != nil && r.Spec.Encryption.Enabled {
		allErrs = append(allErrs, r.validateEncryption(field.NewPath("spec", "encryption"), r.Spec.Encryption)...)
	}
	
	if len(allErrs) == 0 {
		return nil
	}
	
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "backup.yourcompany.com", Kind: "BackupTask"},
		r.Name, allErrs)
}

func (r *BackupTask) validateDatabaseSpec(path *field.Path, spec *DatabaseBackupSpec) field.ErrorList {
	var allErrs field.ErrorList
	
	if spec.Host == "" {
		allErrs = append(allErrs, field.Required(
			path.Child("host"),
			"database host is required",
		))
	}
	
	if spec.Port == nil {
		// 设置默认端口
		switch spec.Type {
		case "mysql":
			port := int32(3306)
			spec.Port = &port
		case "postgresql":
			port := int32(5432)
			spec.Port = &port
		case "mongodb":
			port := int32(27017)
			spec.Port = &port
		case "redis":
			port := int32(6379)
			spec.Port = &port
		default:
			allErrs = append(allErrs, field.Required(
				path.Child("port"),
				"database port is required",
			))
		}
	} else {
		if *spec.Port < 1 || *spec.Port > 65535 {
			allErrs = append(allErrs, field.Invalid(
				path.Child("port"),
				*spec.Port,
				"port must be between 1 and 65535",
			))
		}
	}
	
	if spec.AuthSecretRef.Name == "" {
		allErrs = append(allErrs, field.Required(
			path.Child("authSecretRef", "name"),
			"auth secret name is required",
		))
	}
	
	return allErrs
}

func (r *BackupTask) validateDirectorySpec(path *field.Path, spec *DirectoryBackupSpec) field.ErrorList {
	var allErrs field.ErrorList
	
	if len(spec.Paths) == 0 {
		allErrs = append(allErrs, field.Required(
			path.Child("paths"),
			"at least one directory path is required",
		))
	}
	
	for i, dirPath := range spec.Paths {
		pathPath := path.Child("paths").Index(i)
		
		if dirPath.Path == "" {
			allErrs = append(allErrs, field.Required(
				pathPath.Child("path"),
				"directory path is required",
			))
		}
	}
	
	if spec.CompressionLevel != nil && (*spec.CompressionLevel < 0 || *spec.CompressionLevel > 9) {
		allErrs = append(allErrs, field.Invalid(
			path.Child("compressionLevel"),
			*spec.CompressionLevel,
			"compression level must be between 0 and 9",
		))
	}
	
	return allErrs
}

func (r *BackupTask) validateNFSSpec(path *field.Path, spec *NFSBackupSpec) field.ErrorList {
	var allErrs field.ErrorList
	
	if spec.Server == "" {
		allErrs = append(allErrs, field.Required(
			path.Child("server"),
			"nfs server is required",
		))
	}
	
	if spec.Path == "" {
		allErrs = append(allErrs, field.Required(
			path.Child("path"),
			"nfs path is required",
		))
	}
	
	return allErrs
}

func (r *BackupTask) validateRemoteStorage(path *field.Path, spec *RemoteStorageSpec) field.ErrorList {
	var allErrs field.ErrorList
	
	switch spec.Type {
	case RemoteStorageTypeS3:
		if spec.S3 == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("s3"),
				"s3 configuration is required for s3 storage",
			))
		} else {
			if spec.S3.Endpoint == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("s3", "endpoint"),
					"s3 endpoint is required",
				))
			}
			
			if spec.S3.Bucket == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("s3", "bucket"),
					"s3 bucket is required",
				))
			}
			
			if spec.S3.AuthSecretRef.Name == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("s3", "authSecretRef", "name"),
					"auth secret name is required",
				))
			}
		}
		
	case RemoteStorageTypeSFTP:
		if spec.SFTP == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("sftp"),
				"sftp configuration is required for sftp storage",
			))
		} else {
			if spec.SFTP.Host == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("sftp", "host"),
					"sftp host is required",
				))
			}
			
			if spec.SFTP.Path == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("sftp", "path"),
					"sftp path is required",
				))
			}
			
			if spec.SFTP.AuthSecretRef.Name == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("sftp", "authSecretRef", "name"),
					"auth secret name is required",
				))
			}
		}
		
	case RemoteStorageTypeNFS:
		if spec.NFS == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("nfs"),
				"nfs configuration is required for nfs storage",
			))
		} else {
			if spec.NFS.Server == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("nfs", "server"),
					"nfs server is required",
				))
			}
			
			if spec.NFS.Path == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("nfs", "path"),
					"nfs path is required",
				))
			}
		}
		
	case RemoteStorageTypeSMB:
		if spec.SMB == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("smb"),
				"smb configuration is required for smb storage",
			))
		} else {
			if spec.SMB.Server == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("smb", "server"),
					"smb server is required",
				))
			}
			
			if spec.SMB.Share == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("smb", "share"),
					"smb share is required",
				))
			}
			
			if spec.SMB.AuthSecretRef.Name == "" {
				allErrs = append(allErrs, field.Required(
					path.Child("smb", "authSecretRef", "name"),
					"auth secret name is required",
				))
			}
		}
	}
	
	return allErrs
}

func (r *BackupTask) validateEncryption(path *field.Path, spec *EncryptionSpec) field.ErrorList {
	var allErrs field.ErrorList
	
	if spec.KeySecretRef.Name == "" {
		allErrs = append(allErrs, field.Required(
			path.Child("keySecretRef", "name"),
			"encryption key secret name is required",
		))
	}
	
	return allErrs
}

func isValidCronSchedule(schedule string) bool {
	// 简化的Cron表达式验证
	// 实际应该使用更完整的验证库
	cronRegex := `^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$`
	match, _ := regexp.MatchString(cronRegex, schedule)
	return match
}

func (r *BackupTask) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}