import { Routes } from '@angular/router';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { UploadComponent } from './components/upload/upload.component';
import { DetailsComponent } from './components/details/details.component';
import { authGuard } from './auth.guard';
import { guestGuard } from './guest.guard';

export const routes: Routes = [
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login',    component: LoginComponent,     canActivate: [guestGuard] },
  { path: 'register', component: RegisterComponent,  canActivate: [guestGuard] },
  { path: 'dashboard',      component: DashboardComponent, canActivate: [authGuard] },
  { path: 'upload',         component: UploadComponent,    canActivate: [authGuard] },
  { path: 'details/:id',    component: DetailsComponent,   canActivate: [authGuard] }
];

