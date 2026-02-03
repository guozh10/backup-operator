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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	backupv1alpha1 "backup-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var backuptasklog = logf.Log.WithName("backuptask-resource")

// SetupBackupTaskWebhookWithManager registers the webhook for BackupTask in the manager.
func SetupBackupTaskWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&backupv1alpha1.BackupTask{}).
		WithValidator(&BackupTaskCustomValidator{}).
		WithDefaulter(&BackupTaskCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-backup-mybackup-com-v1alpha1-backuptask,mutating=true,failurePolicy=fail,sideEffects=None,groups=backup.mybackup.com,resources=backuptasks,verbs=create;update,versions=v1alpha1,name=mbackuptask-v1alpha1.kb.io,admissionReviewVersions=v1

// BackupTaskCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind BackupTask when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type BackupTaskCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &BackupTaskCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind BackupTask.
func (d *BackupTaskCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	backuptask, ok := obj.(*backupv1alpha1.BackupTask)

	if !ok {
		return fmt.Errorf("expected an BackupTask object but got %T", obj)
	}
	backuptasklog.Info("Defaulting for BackupTask", "name", backuptask.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-backup-mybackup-com-v1alpha1-backuptask,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup.mybackup.com,resources=backuptasks,verbs=create;update,versions=v1alpha1,name=vbackuptask-v1alpha1.kb.io,admissionReviewVersions=v1

// BackupTaskCustomValidator struct is responsible for validating the BackupTask resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type BackupTaskCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &BackupTaskCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type BackupTask.
func (v *BackupTaskCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	backuptask, ok := obj.(*backupv1alpha1.BackupTask)
	if !ok {
		return nil, fmt.Errorf("expected a BackupTask object but got %T", obj)
	}
	backuptasklog.Info("Validation for BackupTask upon creation", "name", backuptask.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type BackupTask.
func (v *BackupTaskCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	backuptask, ok := newObj.(*backupv1alpha1.BackupTask)
	if !ok {
		return nil, fmt.Errorf("expected a BackupTask object for the newObj but got %T", newObj)
	}
	backuptasklog.Info("Validation for BackupTask upon update", "name", backuptask.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type BackupTask.
func (v *BackupTaskCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	backuptask, ok := obj.(*backupv1alpha1.BackupTask)
	if !ok {
		return nil, fmt.Errorf("expected a BackupTask object but got %T", obj)
	}
	backuptasklog.Info("Validation for BackupTask upon deletion", "name", backuptask.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
