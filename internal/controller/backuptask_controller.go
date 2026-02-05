package controller

import (
	"context"
	"fmt"
	"time"
        "encoding/json"
        "strconv"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	// "sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "backup-operator/api/v1alpha1"
)

//const (
// 备份任务Finalizer
//	backupTaskFinalizer = "backup.mybackup.com/finalizer"

// 标签键
//	backupTaskLabelKey    = "backup.mybackup.com/backup-task"
//	backupTaskUIDLabelKey = "backup.mybackup.com/backup-task-uid"
//	createdByLabelKey     = "backup.mybackup.com/created-by"

// 注解键
//	scheduleAnnotationKey   = "backup.mybackup.com/schedule"
//	lastBackupAnnotationKey = "backup.mybackup.com/last-backup"

// 容器常量
//	backupContainerName = "backup"
//	backupImage         = "guozh10/backup-agent:latest"

// 卷名称
//	backupDataVolumeName = "backup-data"
//	scriptsVolumeName    = "backup-scripts"
//	configVolumeName     = "backup-config"
//	temporaryVolumeName  = "temp"
//)

// BackupTaskReconciler reconciles a BackupTask object
type BackupTaskReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuptasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuptasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuptasks/finalizers,verbs=update
// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuprecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuprecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
func (r *BackupTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("backuptask", req.NamespacedName)
	// 重要：添加日志输出
	log.Info("Reconcile called", "name", req.Name, "namespace", req.Namespace)
	log.Info("开始处理 BackupTask")
	// 获取BackupTask实例
	backupTask := &backupv1alpha1.BackupTask{}
	if err := r.Get(ctx, req.NamespacedName, backupTask); err != nil {
		if errors.IsNotFound(err) {
			// 对象被删除
			log.Error(err, "无法获取 BackupTask")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Info("Processing BackupTask", "schedule", backupTask.Spec.Schedule, "targets", len(backupTask.Spec.Targets))

	// 检查对象是否正在删除
	if !backupTask.ObjectMeta.DeletionTimestamp.IsZero() {
		// 处理Finalizer
		if controllerutil.ContainsFinalizer(backupTask, backupTaskFinalizer) {
			if err := r.finalizeBackupTask(ctx, backupTask); err != nil {
				log.Error(err, "failed to finalize BackupTask")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(backupTask, backupTaskFinalizer)
			if err := r.Update(ctx, backupTask); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 添加Finalizer
	if !controllerutil.ContainsFinalizer(backupTask, backupTaskFinalizer) {
		controllerutil.AddFinalizer(backupTask, backupTaskFinalizer)
		if err := r.Update(ctx, backupTask); err != nil {
			return ctrl.Result{}, err
		}
	}
    	// 创建或更新 ConfigMap
    	configMap, err := r.createTargetConfigMap(backupTask)
    	if err != nil {
        	return ctrl.Result{}, err
    	}
    
    	// 创建 ConfigMap
    	if err := r.Create(ctx, configMap); err != nil {
        	if !errors.IsAlreadyExists(err) {
            	return ctrl.Result{}, err
        	}
        	// 如果已存在，则更新
        	if err := r.Update(ctx, configMap); err != nil {
            	return ctrl.Result{}, err
        	}
    	}

	// 处理备份任务
	result, err := r.reconcileBackupTask(ctx, backupTask)
	if err != nil {
		r.Recorder.Eventf(backupTask, corev1.EventTypeWarning, "ReconcileFailed",
			"Failed to reconcile BackupTask: %v", err)
		log.Error(err, "failed to reconcile BackupTask")
	}

	return result, err

}

func (r *BackupTaskReconciler) reconcileBackupTask(ctx context.Context, backupTask *backupv1alpha1.BackupTask) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("backuptask", backupTask.Name, "namespace", backupTask.Namespace)

	// 1. 创建或更新CronJob
	cronJob, err := r.createOrUpdateCronJob(ctx, backupTask)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create/update CronJob: %w", err)
	}

	// 2. 更新BackupTask状态
	if err := r.updateBackupTaskStatus(ctx, backupTask, cronJob); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
	}

	// 3. 检查是否有正在运行的备份作业
	if cronJob.Status.Active != nil && len(cronJob.Status.Active) > 0 {
		// 有活跃的Job，稍后重试
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// 4. 应用保留策略
	if err := r.applyRetentionPolicy(ctx, backupTask); err != nil {
		log.Error(err, "failed to apply retention policy")
		// 不返回错误，保留策略失败不应阻止下一次备份
	}

	// 5. 计算下一次调度时间
	requeueAfter := r.calculateNextSchedule(backupTask, cronJob)
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *BackupTaskReconciler) createOrUpdateCronJob(ctx context.Context, backupTask *backupv1alpha1.BackupTask) (*batchv1.CronJob, error) {
	cronJobName := fmt.Sprintf("backup-%s", backupTask.Name)

	// 构建CronJob对象
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: backupTask.Namespace,
			Labels: map[string]string{
				backupTaskLabelKey:    backupTask.Name,
				backupTaskUIDLabelKey: string(backupTask.UID),
				createdByLabelKey:     "backup-operator",
			},
		},
	}

	// 创建或更新
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, cronJob, func() error {
		// 设置CronJob规范
		cronJob.Spec = r.buildCronJobSpec(backupTask)

		// 设置OwnerReference
		if err := ctrl.SetControllerReference(backupTask, cronJob, r.Scheme); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	ctrl.Log.Info("CronJob reconciled", "operation", op, "name", cronJobName, "namespace", backupTask.Namespace)
	return cronJob, nil
}

func (r *BackupTaskReconciler) buildCronJobSpec(backupTask *backupv1alpha1.BackupTask) batchv1.CronJobSpec {
	// 使用BackoffLimit防止作业无限重试
	backoffLimit := int32(3)

	return batchv1.CronJobSpec{
		Schedule:                   backupTask.Spec.Schedule,
		StartingDeadlineSeconds:    pointer.Int64(300), // 5分钟
		ConcurrencyPolicy:          batchv1.ForbidConcurrent,
		Suspend:                    pointer.Bool(false),
		SuccessfulJobsHistoryLimit: pointer.Int32(3),
		FailedJobsHistoryLimit:     pointer.Int32(1),
		JobTemplate: batchv1.JobTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					backupTaskLabelKey:    backupTask.Name,
					backupTaskUIDLabelKey: string(backupTask.UID),
					createdByLabelKey:     "backup-operator",
				},
			},
			Spec: batchv1.JobSpec{
				BackoffLimit:            &backoffLimit,
				TTLSecondsAfterFinished: pointer.Int32(86400), // 24小时后删除Job
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							backupTaskLabelKey:    backupTask.Name,
							backupTaskUIDLabelKey: string(backupTask.UID),
							createdByLabelKey:     "backup-operator",
						},
					},
					Spec: r.buildBackupPodSpec(backupTask),
				},
			},
		},
	}
}

func (r *BackupTaskReconciler) buildBackupPodSpec(backupTask *backupv1alpha1.BackupTask) corev1.PodSpec {
	// 从BackupTask中获取Pod模板（如果存在）
	var podSpec corev1.PodSpec
	if backupTask.Spec.PodTemplate != nil && backupTask.Spec.PodTemplate.Spec != nil {
		podSpec = *backupTask.Spec.PodTemplate.Spec
	}

	// 设置默认值
	if podSpec.RestartPolicy == "" {
		podSpec.RestartPolicy = corev1.RestartPolicyOnFailure
	}

	// 添加卷
	podSpec.Volumes = r.buildVolumes(backupTask)

	// 添加容器
	podSpec.Containers = r.buildContainers(backupTask)

	// 设置ServiceAccount（如果需要访问Kubernetes API）
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = "backup-agent"
	}

	// 设置亲和性和容忍度
	if podSpec.Affinity == nil {
		podSpec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
					{
						Weight: 100,
						Preference: corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "node-role.kubernetes.io/backup",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"true"},
								},
							},
						},
					},
				},
			},
		}
	}

	return podSpec
}

func (r *BackupTaskReconciler) buildVolumes(backupTask *backupv1alpha1.BackupTask) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: backupDataVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    corev1.StorageMediumDefault,
					SizeLimit: resource.NewQuantity(10*1024*1024*1024, resource.BinarySI), // 10Gi
				},
			},
		},
		{
			Name: temporaryVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    corev1.StorageMediumDefault,
					SizeLimit: resource.NewQuantity(5*1024*1024*1024, resource.BinarySI), // 5Gi
				},
			},
		},
		{       Name: configVolumeName,
        		VolumeSource: corev1.VolumeSource{
            			ConfigMap: &corev1.ConfigMapVolumeSource{
                			LocalObjectReference: corev1.LocalObjectReference{
                    			Name: r.getTargetConfigMapName(backupTask),
                			},
                			// 可选：指定要挂载的 key
                			Items: []corev1.KeyToPath{
                    				{Key:  "targets.json",Path: "targets.json",},
						{Key:  "remotestorage.json",Path: "remotestorage.json",},
   					},
                			// 可选：设置默认权限
                			DefaultMode: pointer.Int32(0644),
            			},
        		},
		},
	}

	// 为每个需要挂载的PVC添加卷
	for _, target := range backupTask.Spec.Targets {
		if target.Directory != nil {
			for _, dirPath := range target.Directory.Paths {
				if dirPath.PVCRef != nil {
					volumeName := fmt.Sprintf("pvc-%s", dirPath.PVCRef.Name)
					volumes = append(volumes, corev1.Volume{
						Name: volumeName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: dirPath.PVCRef.Name,
								ReadOnly:  true,
							},
						},
					})
				}
			}
		}
	}

	// 添加Secret卷
	if backupTask.Spec.RemoteStorage.SFTP != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "sftp-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: backupTask.Spec.RemoteStorage.SFTP.AuthSecretRef.Name,
				},
			},
		})
	}

	if backupTask.Spec.Encryption != nil && backupTask.Spec.Encryption.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: "encryption-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: backupTask.Spec.Encryption.KeySecretRef.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  "key",
							Path: "encryption.key",
						},
					},
				},
			},
		})
	}

	return volumes
}

func (r *BackupTaskReconciler) getTargetConfigMapName(backupTask *backupv1alpha1.BackupTask) string {
    return fmt.Sprintf("%s-targets", backupTask.Name)
}

func (r *BackupTaskReconciler) getBackupImage(backupTask *backupv1alpha1.BackupTask) string {
	if backupTask.Spec.BackupImage != "" {
		return backupTask.Spec.BackupImage
	}
	return defaultBackupImage // 这里是你原来的常量，可以保留为一个默认值
}

func (r *BackupTaskReconciler) getBackupCmd(backupTask *backupv1alpha1.BackupTask) string {
	if backupTask.Spec.BackupImage != "" {
		return backupTask.Spec.BackupCmd
	}
	return defaultBackupCmd // 这里是你原来的常量，可以保留为一个默认值
}

func (r *BackupTaskReconciler) buildContainers(backupTask *backupv1alpha1.BackupTask) []corev1.Container {
	containers := []corev1.Container{
		{
			Name:            backupContainerName,
			Image:           r.getBackupImage(backupTask),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/bash", "-c", r.getBackupCmd(backupTask)},
			Env:             r.buildEnvVars(backupTask),
			VolumeMounts:    r.buildVolumeMounts(backupTask),
			Resources:       r.buildContainerResources(backupTask),
			// SecurityContext: &corev1.SecurityContext{
			// 	AllowPrivilegeEscalation: pointer.Bool(false),
			//	RunAsNonRoot:             pointer.Bool(true),
			//	RunAsUser:                pointer.Int64(1000),
			//	Capabilities: &corev1.Capabilities{
			//		Drop: []corev1.Capability{"ALL"},
			//	},
			//	SeccompProfile: &corev1.SeccompProfile{
			//		Type: corev1.SeccompProfileTypeRuntimeDefault,
			//	},
			//},
		},
	}
        for c := range containers {
                // 设置默认的 imagePullPolicy
                if containers[c].ImagePullPolicy == "" {
                        containers[c].ImagePullPolicy = corev1.PullAlways
                }
        }
	// 添加sidecar容器（如数据库客户端）
	// for _, target := range backupTask.Spec.Targets {
	// 	if target.Database != nil {
	// 		sidecar := r.buildDatabaseSidecar(target.Database)
	// 		if sidecar != nil {
	// 			containers = append(containers, *sidecar)
	// 		}
	// 	}
	// }

	return containers
}

func (r *BackupTaskReconciler) buildEnvVars(backupTask *backupv1alpha1.BackupTask) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "BACKUP_TASK_NAME",
			Value: backupTask.Name,
		},
		{
			Name:  "BACKUP_TASK_NAMESPACE",
			Value: backupTask.Namespace,
		},
		{
			Name: "BACKUP_ID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	// 添加远程存储配置
	switch backupTask.Spec.RemoteStorage.Type {
	case backupv1alpha1.RemoteStorageTypeS3:
		if backupTask.Spec.RemoteStorage.S3 != nil {
			envVars = append(envVars,
				corev1.EnvVar{
					Name:  "REMOTE_STORAGE_TYPE",
					Value: "s3",
				},
				corev1.EnvVar{
					Name:  "S3_ENDPOINT",
					Value: backupTask.Spec.RemoteStorage.S3.Endpoint,
				},
				corev1.EnvVar{
					Name:  "S3_BUCKET",
					Value: backupTask.Spec.RemoteStorage.S3.Bucket,
				},
				corev1.EnvVar{
					Name:  "S3_PREFIX",
					Value: backupTask.Spec.RemoteStorage.S3.Prefix,
				},
			)
		}
	case backupv1alpha1.RemoteStorageTypeSFTP:
		if backupTask.Spec.RemoteStorage.SFTP != nil {
			envVars = append(envVars,
				corev1.EnvVar{
					Name:  "REMOTE_STORAGE_TYPE",
					Value: "sftp",
				},
				corev1.EnvVar{
					Name:  "SFTP_HOST",
					Value: backupTask.Spec.RemoteStorage.SFTP.Host,
				},
				corev1.EnvVar{
					Name:  "SFTP_PATH",
					Value: backupTask.Spec.RemoteStorage.SFTP.Path,
				},
			)
		}
	}

	// 添加加密配置
	if backupTask.Spec.Encryption != nil && backupTask.Spec.Encryption.Enabled {
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  "ENCRYPTION_ENABLED",
				Value: "true",
			},
			corev1.EnvVar{
				Name:  "ENCRYPTION_ALGORITHM",
				Value: backupTask.Spec.Encryption.Algorithm,
			},
		)
	}

	return envVars
}

func (r *BackupTaskReconciler) buildVolumeMounts(backupTask *backupv1alpha1.BackupTask) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      backupDataVolumeName,
			MountPath: "/backup/data",
		},
		{
			Name:      temporaryVolumeName,
			MountPath: "/tmp",
		},
		{
			Name:     configVolumeName ,
        		MountPath: "/etc/backup/config",
        		ReadOnly:  true,
		},
	}

	// 为每个PVC添加挂载
	for _, target := range backupTask.Spec.Targets {
		if target.Directory != nil {
			for i, dirPath := range target.Directory.Paths {
				if dirPath.PVCRef != nil {
					volumeName := fmt.Sprintf("pvc-%s", dirPath.PVCRef.Name)
					mountPath := fmt.Sprintf("/mnt/pvc-%d", i)
					volumeMounts = append(volumeMounts, corev1.VolumeMount{
						Name:      volumeName,
						MountPath: mountPath,
						SubPath:   dirPath.SubPath,
						ReadOnly:  true,
					})
				}
			}
		}
	}

	return volumeMounts
}

func (r *BackupTaskReconciler) buildContainerResources(backupTask *backupv1alpha1.BackupTask) corev1.ResourceRequirements {
	// 使用BackupTask中指定的资源限制，或使用默认值
	if backupTask.Spec.Resources != nil {
		return *backupTask.Spec.Resources
	}

	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}
}

func (r *BackupTaskReconciler) buildDatabaseSidecar(dbSpec *backupv1alpha1.DatabaseBackupSpec) *corev1.Container {
	switch dbSpec.Type {
	case "mysql":
		return &corev1.Container{
			Name:            "mysql-client",
			Image:           "mysql:8.0",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"sleep", "infinity"},
			Env: []corev1.EnvVar{
				{
					Name:  "MYSQL_HOST",
					Value: dbSpec.Host,
				},
				{
					Name:  "MYSQL_PORT",
					Value: fmt.Sprintf("%d", *dbSpec.Port),
				},
				{
					Name:  "MYSQL_DATABASE",
					Value: dbSpec.Database,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		}
	case "postgresql":
		return &corev1.Container{
			Name:            "postgres-client",
			Image:           "postgres:14",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"sleep", "infinity"},
			Env: []corev1.EnvVar{
				{
					Name:  "PGHOST",
					Value: dbSpec.Host,
				},
				{
					Name:  "PGPORT",
					Value: fmt.Sprintf("%d", *dbSpec.Port),
				},
				{
					Name:  "PGDATABASE",
					Value: dbSpec.Database,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		}
	}
	return nil
}

func (r *BackupTaskReconciler) updateBackupTaskStatus(ctx context.Context, backupTask *backupv1alpha1.BackupTask, cronJob *batchv1.CronJob) error {
	// 获取最新的BackupTask
	latestBackupTask := &backupv1alpha1.BackupTask{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(backupTask), latestBackupTask); err != nil {
		return err
	}

	// 更新状态
	latestBackupTask.Status.ObservedGeneration = backupTask.Generation

	// 设置阶段
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		latestBackupTask.Status.Phase = backupv1alpha1.BackupPhasePending
	} else {
		// 检查是否有活跃的Job
		if len(cronJob.Status.Active) > 0 {
			latestBackupTask.Status.Phase = backupv1alpha1.BackupPhaseRunning
		} else if cronJob.Status.LastScheduleTime != nil {
			latestBackupTask.Status.Phase = backupv1alpha1.BackupPhaseCompleted
		} else {
			latestBackupTask.Status.Phase = backupv1alpha1.BackupPhasePending
		}
	}

	// 更新调度时间
	if cronJob.Status.LastScheduleTime != nil {
		latestBackupTask.Status.LastScheduledTime = cronJob.Status.LastScheduleTime
	}

	// 更新活跃的备份Job
	if len(cronJob.Status.Active) > 0 {
		latestBackupTask.Status.ActiveBackupJob = &corev1.ObjectReference{
			APIVersion: "batch/v1",
			Kind:       "Job",
			Name:       cronJob.Status.Active[0].Name,
			Namespace:  cronJob.Namespace,
		}
	} else {
		latestBackupTask.Status.ActiveBackupJob = nil
	}

	// 计算下一个调度时间
	// if cronJob.Status.NextScheduleTime != nil {
	// 	latestBackupTask.Status.NextScheduleTime = cronJob.Status.NextScheduleTime
	// }

	// 更新条件
	r.updateConditions(latestBackupTask)

	// 更新状态
	return r.Status().Update(ctx, latestBackupTask)
}

func (r *BackupTaskReconciler) updateConditions(backupTask *backupv1alpha1.BackupTask) {
	now := metav1.Now()

	// 调度条件
	scheduledCondition := backupv1alpha1.BackupTaskCondition{
		Type:               backupv1alpha1.BackupTaskScheduled,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: now,
	}

	if backupTask.Status.LastScheduledTime == nil {
		scheduledCondition.Status = corev1.ConditionFalse
		scheduledCondition.Reason = "NotScheduled"
		scheduledCondition.Message = "Backup task has not been scheduled yet"
	}

	// 运行条件
	runningCondition := backupv1alpha1.BackupTaskCondition{
		Type:               backupv1alpha1.BackupTaskRunning,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: now,
	}

	if backupTask.Status.Phase == backupv1alpha1.BackupPhaseRunning {
		runningCondition.Status = corev1.ConditionTrue
		runningCondition.Reason = "BackupInProgress"
		runningCondition.Message = "Backup is currently running"
	}

	// 更新条件列表
	backupTask.Status.Conditions = []backupv1alpha1.BackupTaskCondition{
		scheduledCondition,
		runningCondition,
	}
}

func (r *BackupTaskReconciler) applyRetentionPolicy(ctx context.Context, backupTask *backupv1alpha1.BackupTask) error {
	if backupTask.Spec.Retention == nil {
		return nil
	}

	// 获取关联的BackupRecord列表
	backupRecords := &backupv1alpha1.BackupRecordList{}
	labelSelector := map[string]string{
		backupTaskLabelKey: backupTask.Name,
	}

	if err := r.List(ctx, backupRecords, client.InNamespace(backupTask.Namespace), client.MatchingLabels(labelSelector)); err != nil {
		return fmt.Errorf("failed to list BackupRecords: %w", err)
	}

	// 应用保留策略
	recordsToDelete := r.filterRecordsToDelete(backupRecords.Items, backupTask.Spec.Retention)

	for _, record := range recordsToDelete {
		if err := r.Delete(ctx, &record); err != nil {
			r.Log.Error(err, "failed to delete old BackupRecord", "name", record.Name)
			continue
		}
		ctrl.Log.Info("Deleted old BackupRecord due to retention policy", "name", record.Name, "backupTask", backupTask.Name)
	}

	// 更新状态
	backupTask.Status.BackupHistoryCount = int32(len(backupRecords.Items) - len(recordsToDelete))

	return nil
}

func (r *BackupTaskReconciler) filterRecordsToDelete(records []backupv1alpha1.BackupRecord, retention *backupv1alpha1.RetentionPolicy) []backupv1alpha1.BackupRecord {
	var toDelete []backupv1alpha1.BackupRecord
	now := time.Now()

	// 按创建时间排序（最新的在前）
	sortedRecords := make([]backupv1alpha1.BackupRecord, len(records))
	copy(sortedRecords, records)

	// 应用最大备份数量限制
	if retention.MaxBackupCount != nil && len(sortedRecords) > int(*retention.MaxBackupCount) {
		toDelete = append(toDelete, sortedRecords[*retention.MaxBackupCount:]...)
	}

	// 应用最大年龄限制
	if retention.MaxAgeDays != nil {
		maxAge := time.Duration(*retention.MaxAgeDays) * 24 * time.Hour
		for _, record := range sortedRecords {
			if record.CreationTimestamp.Add(maxAge).Before(now) {
				toDelete = append(toDelete, record)
			}
		}
	}

	return toDelete
}

func (r *BackupTaskReconciler) calculateNextSchedule(backupTask *backupv1alpha1.BackupTask, cronJob *batchv1.CronJob) time.Duration {
	return 5 * time.Minute
}

func (r *BackupTaskReconciler) finalizeBackupTask(ctx context.Context, backupTask *backupv1alpha1.BackupTask) error {
	// 删除关联的CronJob
	cronJobName := fmt.Sprintf("backup-%s", backupTask.Name)
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: backupTask.Namespace,
		},
	}

	if err := r.Delete(ctx, cronJob); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete CronJob: %w", err)
	}

	// 删除关联的BackupRecords
	backupRecords := &backupv1alpha1.BackupRecordList{}
	labelSelector := map[string]string{
		backupTaskLabelKey: backupTask.Name,
	}

	if err := r.List(ctx, backupRecords, client.InNamespace(backupTask.Namespace), client.MatchingLabels(labelSelector)); err != nil {
		return fmt.Errorf("failed to list BackupRecords: %w", err)
	}

	for _, record := range backupRecords.Items {
		if err := r.Delete(ctx, &record); err != nil {
			ctrl.Log.Error(err, "failed to delete BackupRecord", "name", record.Name)
		}
	}

	ctrl.Log.Info("Successfully finalized BackupTask", "name", backupTask.Name)
	return nil
}
// createTargetConfigMap
func (r *BackupTaskReconciler) createTargetConfigMap(backupTask *backupv1alpha1.BackupTask) (*corev1.ConfigMap, error) {
    configMap := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-targets", backupTask.Name),
            Namespace: backupTask.Namespace,
            Labels: map[string]string{
                "app":        "backup-operator",
                "backuptask": backupTask.Name,
            },
        },
        Data: make(map[string]string),
    }

    // 序列化每个 target 为单独的 JSON 文件
    // for i, target := range backupTask.Spec.Targets {
    //    targetJSON, err := json.MarshalIndent(target, "", "  ")
    //    if err != nil {
            // 记录错误但继续处理其他 targets
    //        r.Log.Error(err, "Failed to marshal target", "target", target.Name)
    //        continue
    //    }
    //    configMap.Data[fmt.Sprintf("target-%d.json", i)] = string(targetJSON)
    //}

    // 同时存储整个 targets 数组
    allTargetsJSON, err := json.MarshalIndent(backupTask.Spec.Targets, "", "  ")
    if err != nil {
        return nil, err
    }
    configMap.Data["targets.json"] = string(allTargetsJSON)

    allRemoteStorageJSON, err := json.MarshalIndent(backupTask.Spec.RemoteStorage, "", "  ")
    if err != nil {
        return nil, err
    }
    configMap.Data["remotestorage.json"] = string(allRemoteStorageJSON)
    
    // 添加一些元数据
    configMap.Data["target-count"] = strconv.Itoa(len(backupTask.Spec.Targets))
    // configMap.Data["backup-task-name"] = backupTask.Name

    return configMap, nil
}


// SetupWithManager sets up the controller with the Manager.
func (r *BackupTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupTask{}).
		Owns(&batchv1.CronJob{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
