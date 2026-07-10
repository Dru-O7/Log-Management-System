import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './services/auth.service';

/** Protects guest-only routes (login, register).
 *  Redirects already-authenticated users to /dashboard. */
export const guestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (!auth.getCurrentUser()) {
    return true;
  }
  return router.createUrlTree(['/dashboard']);
};
