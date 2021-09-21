import React from 'react';
import { 
    Hero 
} from 'components';
import { MissionSlide } from 'components/slides';

function HomePage() {
    return (
        <div>
            <Hero text="Your portal to idea monetization" />
            <MissionSlide />
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };