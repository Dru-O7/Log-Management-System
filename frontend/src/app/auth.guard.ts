import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './services/auth.service';

/** Protects routes that require authentication.
 *  Redirects to /login if the user is not logged in. */
export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (auth.getCurrentUser()) {
    return true;
  }
  return router.createUrlTree(['/login']);
};
