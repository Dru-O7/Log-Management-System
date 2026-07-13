import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from './services/auth.service';

// Protects /admin — allows "Admin" role only
export const adminGuard = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const user = auth.getCurrentUser();

  if (user && (user.Role === 'Admin' || user.Role === 'SuperAdmin')) {
    return true;
  }

  router.navigate(['/dashboard']);
  return false;
};

