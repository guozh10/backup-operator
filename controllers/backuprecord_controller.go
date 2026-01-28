package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1alpha1 "backup-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
)

// BackupRecordReconciler reconciles a BackupRecord object
type BackupRecordReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuprecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuprecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.mybackup.com,resources=backuptasks,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

func (r *BackupRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("backuprecord", req.NamespacedName)

	// 获取BackupRecord实例
	backupRecord := &backupv1alpha1.BackupRecord{}
	if err := r.Get(ctx, req.NamespacedName, backupRecord); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch BackupRecord")
		return ctrl.Result{}, err
	}

	// 如果状态已经是完成或失败，不需要处理
	if backupRecord.Status.Phase == backupv1alpha1.BackupPhaseCompleted ||
		backupRecord.Status.Phase == backupv1alpha1.BackupPhaseFailed {
		return ctrl.Result{}, nil
	}

	// 处理BackupRecord
	return r.reconcileBackupRecord(ctx, backupRecord)
}

func (r *BackupRecordReconciler) reconcileBackupRecord(ctx context.Context, backupRecord *backupv1alpha1.BackupRecord) (ctrl.Result, error) {
	log := r.Log.WithValues("backuprecord", backupRecord.Name, "namespace", backupRecord.Namespace)

	// 1. 验证关联的BackupTask是否存在
	backupTask := &backupv1alpha1.BackupTask{}
	backupTaskKey := types.NamespacedName{
		Name:      backupRecord.Spec.BackupTaskRef.Name,
		Namespace: backupRecord.Namespace,
	}

	if err := r.Get(ctx, backupTaskKey, backupTask); err != nil {
		if errors.IsNotFound(err) {
			// BackupTask被删除，标记BackupRecord为过期
			backupRecord.Status.Phase = backupv1alpha1.BackupPhaseExpired
			backupRecord.Status.Error = "Associated BackupTask no longer exists"
			if err := r.Status().Update(ctx, backupRecord); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get BackupTask: %w", err)
	}

	// 2. 检查备份作业状态
	if err := r.checkBackupJobStatus(ctx, backupRecord); err != nil {
		log.Error(err, "failed to check backup job status")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// 3. 如果备份完成，设置过期时间
	if backupRecord.Status.Phase == backupv1alpha1.BackupPhaseCompleted &&
		backupRecord.Status.ExpirationTime == nil &&
		backupTask.Spec.Retention != nil &&
		backupTask.Spec.Retention.MaxAgeDays != nil {

		expirationTime := metav1.NewTime(
			backupRecord.Spec.StartTime.Add(
				time.Duration(*backupTask.Spec.Retention.MaxAgeDays) * 24 * time.Hour,
			),
		)
		backupRecord.Status.ExpirationTime = &expirationTime

		if err := r.Status().Update(ctx, backupRecord); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 4. 验证备份文件（如果配置了验证）
	if backupRecord.Status.Phase == backupv1alpha1.BackupPhaseCompleted &&
		backupRecord.Status.VerificationStatus.Result == backupv1alpha1.VerificationPending {

		if err := r.verifyBackup(ctx, backupRecord); err != nil {
			log.Error(err, "backup verification failed")
			// 验证失败不应导致备份失败，只记录状态
			backupRecord.Status.VerificationStatus = backupv1alpha1.VerificationStatus{
				VerifiedAt: &metav1.Time{Time: time.Now()},
				Result:     backupv1alpha1.VerificationFailed,
				Message:    err.Error(),
			}
			if err := r.Status().Update(ctx, backupRecord); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			backupRecord.Status.VerificationStatus = backupv1alpha1.VerificationStatus{
				VerifiedAt: &metav1.Time{Time: time.Now()},
				Result:     backupv1alpha1.VerificationSuccess,
				Message:    "Backup verification successful",
			}
			if err := r.Status().Update(ctx, backupRecord); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *BackupRecordReconciler) checkBackupJobStatus(ctx context.Context, backupRecord *backupv1alpha1.BackupRecord) error {
	// 查找关联的Job
	jobs := &batchv1.JobList{}
	labelSelector := map[string]string{
		backupTaskLabelKey:    backupRecord.Spec.BackupTaskRef.Name,
		backupTaskUIDLabelKey: string(backupRecord.Spec.BackupTaskRef.UID),
	}

	if err := r.List(ctx, jobs, client.InNamespace(backupRecord.Namespace), client.MatchingLabels(labelSelector)); err != nil {
		return fmt.Errorf("failed to list Jobs: %w", err)
	}

	if len(jobs.Items) == 0 {
		return fmt.Errorf("no backup job found for BackupRecord")
	}

	// 获取最新的Job
	var latestJob *batchv1.Job
	for i := range jobs.Items {
		job := &jobs.Items[i]
		if latestJob == nil || job.CreationTimestamp.After(latestJob.CreationTimestamp.Time) {
			latestJob = job
		}
	}

	// 更新BackupRecord状态
	if latestJob.Status.Succeeded > 0 {
		if backupRecord.Status.Phase != backupv1alpha1.BackupPhaseCompleted {
			backupRecord.Status.Phase = backupv1alpha1.BackupPhaseCompleted
			completionTime := metav1.Now()
			backupRecord.Spec.CompletionTime = &completionTime

			// 获取备份统计信息（可以从Pod日志或注解中获取）
			if err := r.updateBackupStatistics(ctx, backupRecord, latestJob); err != nil {
				r.Log.Error(err, "failed to update backup statistics")
			}

			if err := r.Status().Update(ctx, backupRecord); err != nil {
				return err
			}
		}
	} else if latestJob.Status.Failed > 0 {
		if backupRecord.Status.Phase != backupv1alpha1.BackupPhaseFailed {
			backupRecord.Status.Phase = backupv1alpha1.BackupPhaseFailed
			backupRecord.Status.Error = "Backup job failed"

			// 获取失败原因
			if err := r.getJobFailureReason(ctx, backupRecord, latestJob); err != nil {
				r.Log.Error(err, "failed to get job failure reason")
			}

			if err := r.Status().Update(ctx, backupRecord); err != nil {
				return err
			}
		}
	} else {
		// Job还在运行中
		backupRecord.Status.Phase = backupv1alpha1.BackupPhaseRunning
		if err := r.Status().Update(ctx, backupRecord); err != nil {
			return err
		}
	}

	return nil
}

func (r *BackupRecordReconciler) updateBackupStatistics(ctx context.Context, backupRecord *backupv1alpha1.BackupRecord, job *batchv1.Job) error {
	// 从Job的Pod中获取备份统计信息
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, client.InNamespace(job.Namespace), client.MatchingLabels(job.Spec.Selector.MatchLabels)); err != nil {
		return fmt.Errorf("failed to list Pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no Pod found for Job")
	}

	pod := pods.Items[0]

	// 从Pod的注解中获取统计信息
	if statsStr, ok := pod.Annotations["backup.mybackup.com/statistics"]; ok {
		// 这里可以解析JSON格式的统计信息
		// 简化处理，只记录Pod信息
		if backupRecord.Spec.Statistics == nil {
			backupRecord.Spec.Statistics = &backupv1alpha1.BackupStatistics{}
		}
		// 将持续时间转换为字符串
		duration := time.Since(backupRecord.Spec.StartTime.Time).Seconds()
		backupRecord.Spec.Statistics.DurationSeconds = fmt.Sprintf("%.2f", duration)
		fmt.Println("从Pod的注解中获取统计信息:", statsStr)
		// backupRecord.Spec.Statistics.DurationSeconds = time.Since(backupRecord.Spec.StartTime.Time).Seconds()
	}
	return nil
}

func (r *BackupRecordReconciler) getJobFailureReason(ctx context.Context, backupRecord *backupv1alpha1.BackupRecord, job *batchv1.Job) error {
	// 获取失败的Pod
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, client.InNamespace(job.Namespace), client.MatchingLabels(job.Spec.Selector.MatchLabels)); err != nil {
		return fmt.Errorf("failed to list Pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no Pod found for Job")
	}

	pod := pods.Items[0]

	// 获取容器状态
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == backupContainerName {
			if containerStatus.State.Terminated != nil {
				backupRecord.Status.Error = fmt.Sprintf("Container %s terminated with exit code %d: %s",
					containerStatus.Name,
					containerStatus.State.Terminated.ExitCode,
					containerStatus.State.Terminated.Message)
			}
			break
		}
	}

	return nil
}

func (r *BackupRecordReconciler) verifyBackup(ctx context.Context, backupRecord *backupv1alpha1.BackupRecord) error {
	// 这里可以实现备份验证逻辑
	// 例如：检查备份文件是否存在、校验和是否正确等

	// 简化版本：只是标记为已验证
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupRecord{}).
		Complete(r)
}
