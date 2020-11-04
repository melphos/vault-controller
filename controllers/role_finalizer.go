package controllers

import (
	"context"

	apiv1 "redoute.io/api/vault/vault-controller/api/v1"
)

func (r *RoleReconciler) addFinalizer(instance *apiv1.Role) error {
	instance.AddFinalizer(apiv1.RoleFinalizer)
	return r.Update(context.Background(), instance)
}

func (r *RoleReconciler) handleFinalizer(s *apiv1.Role) error {
	if !s.HasFinalizer(apiv1.RoleFinalizer) {
		return nil
	}

	if err := r.delete(s); err != nil {
		return err
	}
	s.RemoveFinalizer(apiv1.RoleFinalizer)
	return r.Update(context.Background(), s)
}
