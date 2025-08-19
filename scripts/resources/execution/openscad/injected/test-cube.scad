// Simple parametric cube example
cube_size = 20;
sphere_radius = 5;

difference() {
    // Main cube
    cube([cube_size, cube_size, cube_size], center = true);
    
    // Subtract spheres from each face center
    sphere(r = sphere_radius);
    translate([0, 0, cube_size/2]) sphere(r = sphere_radius/2);
    translate([0, 0, -cube_size/2]) sphere(r = sphere_radius/2);
    translate([cube_size/2, 0, 0]) sphere(r = sphere_radius/2);
    translate([-cube_size/2, 0, 0]) sphere(r = sphere_radius/2);
    translate([0, cube_size/2, 0]) sphere(r = sphere_radius/2);
    translate([0, -cube_size/2, 0]) sphere(r = sphere_radius/2);
}