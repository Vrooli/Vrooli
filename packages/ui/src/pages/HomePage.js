import React from 'react';
import { 
    FeaturedProducts,
    Hero 
} from 'components';

function HomePage() {
    return (
        <div>
            <Hero text="Your portal to idea monetization" />
            <FeaturedProducts />
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };