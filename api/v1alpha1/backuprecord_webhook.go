package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupWebhookWithManager 设置webhook管理器
func (r *BackupRecord) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}
