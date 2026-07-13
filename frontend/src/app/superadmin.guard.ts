import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from './services/auth.service';

// Protects /superadmin — allows "SuperAdmin" role only
export const superAdminGuard = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const user = auth.getCurrentUser();

  if (user && user.Role === 'SuperAdmin') {
    return true;
  }

  router.navigate(['/dashboard']);
  return false;
};
